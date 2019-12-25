package lbctrl

import (
	"context"
	"reflect"
	"sync"

	"github.com/zdnscloud/elb-controller/driver"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/helper"
	"github.com/zdnscloud/gok8s/predicate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	taskBufferCount = 30
	maxTaskFailures = 5

	ZcloudLBServiceFinalizer = "lb.zcloud.cn/protect"
)

type LBControlManager struct {
	clusterName string
	client      client.Client
	driver      driver.Driver
	taskCh      chan Task
	stopCh      chan struct{}
	nodes       map[string]string
	lock        sync.Mutex
}

func New(cli client.Client, cache cache.Cache, clusterName string, lbDriver driver.Driver) (*LBControlManager, error) {
	ctrl := controller.New("elb-controller", cache, scheme.Scheme)
	ctrl.Watch(&corev1.Endpoints{})
	ctrl.Watch(&corev1.Service{})
	ctrl.Watch(&corev1.Node{})

	nodes, err := getNodeIPMap(cli)
	if err != nil {
		return nil, err
	}

	m := &LBControlManager{
		clusterName: clusterName,
		client:      cli,
		driver:      lbDriver,
		taskCh:      make(chan Task, taskBufferCount),
		stopCh:      make(chan struct{}),
		nodes:       nodes,
	}

	go ctrl.Start(m.stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	go m.loop()
	return m, nil
}

func (m *LBControlManager) loop() {
	for {
		t := <-m.taskCh
		if isTaskFailureExceed(t) {
			continue
		}
		switch t.Type {
		case CreateTask:
			m.handleCreateTask(t)
		case UpdateTask:
			m.handleUpdateTask(t)
		case DeleteTask:
			m.handleDeleteTask(t)
		default:
			log.Warnf("[TaskLoop] unknown task type %s", t.Type)
		}
	}
}

func isTaskFailureExceed(t Task) bool {
	if t.Failures == maxTaskFailures {
		log.Warnf("[TaskLoop] drop task %s due to exceed max task failures %v", t.ToJson(), maxTaskFailures)
		return true
	}
	return false
}

func (m *LBControlManager) handleCreateTask(t Task) {
	if err := m.driver.Create(*t.NewConfig); err != nil {
		log.Warnf("[TaskLoop] task %s failed %s", t.ToJson(), err.Error())
		m.handleFailedTask(t)
		return
	}
	log.Debugf("[TaskLoop] task %s succeed", t.ToJson())
	if err := addSvcFinalizerAndUpdateStatus(m.client, *t.NewConfig); err != nil {
		log.Warnf("[TaskLoop] add service finalizer or update status failed %s", err.Error())
		m.handleFailedTask(t)
		return
	}
	if err := addEpFinalizer(m.client, *t.NewConfig); err != nil {
		log.Warnf("[TaskLoop] add endpoints finalizer failed %s", err.Error())
		m.handleFailedTask(t)
		return
	}
}

func (m *LBControlManager) handleUpdateTask(t Task) {
	if err := m.driver.Update(*t.OldConfig, *t.NewConfig); err != nil {
		log.Warnf("[TaskLoop] task %s failed %s", t.ToJson(), err.Error())
		m.handleFailedTask(t)
		return
	}
	log.Debugf("[TaskLoop] task %s succeed", t.ToJson())
	if t.OldConfig.VIP == t.NewConfig.VIP {
		return
	}
	if err := addSvcFinalizerAndUpdateStatus(m.client, *t.NewConfig); err != nil {
		log.Warnf("[TaskLoop] add service finalizer or update status failed %s", err.Error())
	}
}

func (m *LBControlManager) handleDeleteTask(t Task) {
	if err := m.driver.Delete(*t.NewConfig); err != nil {
		log.Warnf("[TaskLoop] task %s failed %s", t.ToJson(), err.Error())
		m.handleFailedTask(t)
		return
	}
	log.Debugf("[TaskLoop] task %s succeed", t.ToJson())
	if err := removeFinalizer(m.client, *t.NewConfig); err != nil {
		log.Warnf("[TaskLoop] remove finalizer failed %s", err.Error())
		m.handleFailedTask(t)
	}
}

func (m *LBControlManager) handleFailedTask(t Task) {
	t.Failures += 1
	m.taskCh <- t
}

func addSvcFinalizerAndUpdateStatus(cli client.Client, config driver.Config) error {
	svc := &corev1.Service{}
	if err := cli.Get(context.TODO(), types.NamespacedName{Namespace: config.K8sNamespace, Name: config.K8sService}, svc); err != nil {
		return err
	}

	helper.AddFinalizer(svc, ZcloudLBServiceFinalizer)
	svc.Status.LoadBalancer = corev1.LoadBalancerStatus{
		Ingress: []corev1.LoadBalancerIngress{
			corev1.LoadBalancerIngress{
				IP: config.VIP,
			},
		},
	}
	if err := cli.Update(context.TODO(), svc); err != nil {
		return err
	}
	return cli.Status().Update(context.TODO(), svc)
}

func addEpFinalizer(cli client.Client, config driver.Config) error {
	ep := &corev1.Endpoints{}
	if err := cli.Get(context.TODO(), types.NamespacedName{Namespace: config.K8sNamespace, Name: config.K8sService}, ep); err != nil {
		return err
	}

	helper.AddFinalizer(ep, ZcloudLBServiceFinalizer)
	return cli.Update(context.TODO(), ep)
}

func removeFinalizer(cli client.Client, config driver.Config) error {
	svc := &corev1.Service{}
	if err := cli.Get(context.TODO(), types.NamespacedName{Namespace: config.K8sNamespace, Name: config.K8sService}, svc); err != nil {
		return err
	}
	helper.RemoveFinalizer(svc, ZcloudLBServiceFinalizer)
	if err := cli.Update(context.TODO(), svc); err != nil {
		return err
	}

	ep := &corev1.Endpoints{}
	if err := cli.Get(context.TODO(), types.NamespacedName{Namespace: config.K8sNamespace, Name: config.K8sService}, ep); err != nil {
		return err
	}
	helper.RemoveFinalizer(ep, ZcloudLBServiceFinalizer)
	return cli.Update(context.TODO(), ep)
}

func (m *LBControlManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.Endpoints:
		m.onCreateEndpoints(obj)
	case *corev1.Node:
		m.onCreateNode(obj)
	}
	return handler.Result{}, nil
}

func (m *LBControlManager) onCreateEndpoints(ep *corev1.Endpoints) {
	svc := &corev1.Service{}
	if err := m.client.Get(context.TODO(), types.NamespacedName{Namespace: ep.Namespace, Name: ep.Name}, svc); err != nil {
		log.Warnf("[Event] get endpoints %s service failed %s", genObjNamespacedName(ep.Namespace, ep.Name), err.Error())
		return
	}
	if !isServiceNeedHandle(svc) {
		return
	}
	if svc.ObjectMeta.DeletionTimestamp != nil {
		log.Debugf("[Event] service %s DeletionTimestamp not nil, will delete", genObjNamespacedName(svc.Namespace, svc.Name))
		m.onDeleteService(svc)
		return
	}
	log.Debugf("[Event] service %s created", genObjNamespacedName(svc.Namespace, svc.Name))
	config := genLBConfig(svc, ep, m.clusterName, m.nodes)
	m.taskCh <- NewTask(CreateTask, nil, &config)
}

func (m *LBControlManager) onCreateNode(n *corev1.Node) {
	log.Debugf("[Event] node %s created", n.Name)
	m.lock.Lock()
	defer m.lock.Unlock()
	m.nodes[n.Name] = n.Status.Addresses[0].Address
}

func (m *LBControlManager) onDeleteService(s *corev1.Service) {
	if !isServiceNeedHandle(s) {
		return
	}
	ep := &corev1.Endpoints{}
	if err := m.client.Get(context.TODO(), types.NamespacedName{Namespace: s.Namespace, Name: s.Name}, ep); err != nil {
		log.Warnf("[Event] get service %s endpoints failed %s", genObjNamespacedName(s.Namespace, s.Name), err.Error())
		return
	}
	log.Debugf("[Event] service %s deleted", genObjNamespacedName(s.Namespace, s.Name))
	config := genLBConfig(s, ep, m.clusterName, m.nodes)
	m.taskCh <- NewTask(DeleteTask, nil, &config)
}

func (m *LBControlManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch new := e.ObjectNew.(type) {
	case *corev1.Service:
		old := e.ObjectOld.(*corev1.Service)
		m.onUpdateService(old, new)
	case *corev1.Endpoints:
		old := e.ObjectOld.(*corev1.Endpoints)
		m.onUpdateEndpoints(old, new)
	}
	return handler.Result{}, nil
}

func (m *LBControlManager) onUpdateService(old, new *corev1.Service) {
	if !isServiceNeedHandle(new) {
		return
	}

	if new.ObjectMeta.DeletionTimestamp != nil {
		log.Debugf("[Event] service %s DeletionTimestamp is not nil, will delete", genObjNamespacedName(new.Namespace, new.Name))
		m.onDeleteService(new)
		return
	}
	if reflect.DeepEqual(old.ObjectMeta.Annotations, new.ObjectMeta.Annotations) && reflect.DeepEqual(old.Spec, new.Spec) {
		return
	}

	log.Debugf("[Event] service %s updated", genObjNamespacedName(new.Namespace, new.Name))
	ep := &corev1.Endpoints{}
	if err := m.client.Get(context.TODO(), types.NamespacedName{Namespace: new.Namespace, Name: new.Name}, ep); err != nil {
		log.Warnf("[Event] get service %s endpoints failed %s", genObjNamespacedName(new.Namespace, new.Name), err.Error())
		return
	}

	oldConfig := genLBConfig(old, ep, m.clusterName, m.nodes)
	newConfig := genLBConfig(new, ep, m.clusterName, m.nodes)
	m.taskCh <- NewTask(UpdateTask, &oldConfig, &newConfig)
}

func (m *LBControlManager) onUpdateEndpoints(old, new *corev1.Endpoints) {
	if reflect.DeepEqual(old.Subsets, new.Subsets) {
		return
	}

	log.Debugf("[Event] endpoints %s updated", genObjNamespacedName(new.Namespace, new.Name))
	svc := &corev1.Service{}
	if err := m.client.Get(context.TODO(), types.NamespacedName{Namespace: new.Namespace, Name: new.Name}, svc); err != nil {
		log.Warnf("[Event] get endpoints %s service failed %s", genObjNamespacedName(new.Namespace, new.Name), err.Error())
		return
	}

	oldConfig := genLBConfig(svc, old, m.clusterName, m.nodes)
	newConfig := genLBConfig(svc, new, m.clusterName, m.nodes)
	m.taskCh <- NewTask(UpdateTask, &oldConfig, &newConfig)
}

func (m *LBControlManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.Node:
		m.onDeleteNode(obj)
	}
	return handler.Result{}, nil
}

func (m *LBControlManager) onDeleteNode(n *corev1.Node) {
	log.Debugf("[Event] node %s deleted", n.Name)
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.nodes[n.Name]
	if !ok {
		log.Warnf("node %s doesn't exist in cache", n.Name)
		return
	}
	delete(m.nodes, n.Name)
}

func (m *LBControlManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}
