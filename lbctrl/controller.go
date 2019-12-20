package lbctrl

import (
	"context"
	"fmt"
	"sync"

	"github.com/zdnscloud/elb-controller/driver"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/controller"
	"github.com/zdnscloud/gok8s/event"
	"github.com/zdnscloud/gok8s/handler"
	"github.com/zdnscloud/gok8s/predicate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	taskBufferCount = 30
	maxTaskFailures = 3

	zcloudLBServiceFinalizer = "lb.zcloud.cn/protect"
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

	nodeIpMap, err := getNodeIPMap(cli)
	if err != nil {
		return nil, err
	}

	m := &LBControlManager{
		clusterName: clusterName,
		client:      cli,
		driver:      lbDriver,
		taskCh:      make(chan Task, taskBufferCount),
		stopCh:      make(chan struct{}),
		nodes:       nodeIpMap,
	}

	go ctrl.Start(m.stopCh, m, predicate.NewIgnoreUnchangedUpdate())
	return m, nil
}

func (m *LBControlManager) Loop() {
	for {
		t := <-m.taskCh

		if t.Failures == maxTaskFailures {
			log.Warnf("[TaskLoop] drop task %s due to exceed max task failures %v", t.ToJson(), maxTaskFailures)
			continue
		}
		switch t.Type {
		case CreateTask:
			if err := m.driver.Create(t.NewConfig); err != nil {
				log.Warnf("[TaskLoop] create task %s failed %s", t.ToJson(), err.Error())
				m.readdTaskWhenFailed(t)
				continue
			}
			log.Debugf("[TaskLoop] create task %s succeed", t.ToJson())
			if err := updateServiceStatus(m.client, t.NewConfig); err != nil {
				log.Warnf("[TaskLoop] update service status failed %s", err.Error())
			}
		case UpdateTask:
			if err := m.driver.Update(t.OldConfig, t.NewConfig); err != nil {
				log.Warnf("[TaskLoop] update task %s failed %s", t.ToJson(), err.Error())
				m.readdTaskWhenFailed(t)
				continue
			}
			log.Debugf("[TaskLoop] update task %s succeed", t.ToJson())
			if t.OldConfig.VIP == t.NewConfig.VIP {
				continue
			}
			if err := updateServiceStatus(m.client, t.NewConfig); err != nil {
				log.Warnf("[TaskLoop] update service status failed %s", err.Error())
			}
		case DeleteTask:
			if err := m.driver.Delete(t.NewConfig); err != nil {
				log.Warnf("[TaskLoop] delete task %s failed %s", t.ToJson(), err.Error())
				m.readdTaskWhenFailed(t)
			}
			log.Debugf("[TaskLoop] delete task %s succeed", t.ToJson())
			if err := removeZcloudFinalizerIfNeed(m.client, t.NewConfig); err != nil {
				log.Warnf("[TaskLoop] remove service finalizer failed %s", err.Error())
			}
		default:
			log.Warnf("[TaskLoop] unknown task type %s", t.Type)
		}
	}
}

func (m *LBControlManager) readdTaskWhenFailed(t Task) {
	t.Failures += 1
	m.taskCh <- t
}

func updateServiceStatus(cli client.Client, config driver.Config) error {
	svc := &corev1.Service{}
	if err := cli.Get(context.TODO(), types.NamespacedName{Namespace: config.K8sNamespace, Name: config.K8sService}, svc); err != nil {
		return err
	}
	svc.Status.LoadBalancer = corev1.LoadBalancerStatus{
		Ingress: []corev1.LoadBalancerIngress{
			corev1.LoadBalancerIngress{
				IP: config.VIP,
			},
		},
	}
	return cli.Status().Update(context.TODO(), svc)
}

func removeZcloudFinalizerIfNeed(cli client.Client, config driver.Config) error {
	svc := &corev1.Service{}
	if err := cli.Get(context.TODO(), types.NamespacedName{Namespace: config.K8sNamespace, Name: config.K8sService}, svc); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	var found bool
	newFinalizers := []string{}
	for _, f := range svc.Finalizers {
		if f == zcloudLBServiceFinalizer {
			found = true
		} else {
			newFinalizers = append(newFinalizers, f)
		}
	}

	if !found {
		return nil
	}
	svc.Finalizers = newFinalizers
	return cli.Update(context.TODO(), svc)
}

func (m *LBControlManager) OnCreate(e event.CreateEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.Service:
		if isServiceNeedHandle(obj) {
			if obj.ObjectMeta.DeletionTimestamp != nil {
				m.addDeleteTask(obj)
			} else {
				log.Debugf("[Event] service %s created", genOBjNamespacedName(obj.Namespace, obj.Name))
				config, err := genLBConfigByService(m.client, obj, m.clusterName, m.nodes)
				if err != nil {
					log.Warnf("[Event] created-service %s config gen failed %s", genOBjNamespacedName(obj.Namespace, obj.Name), err.Error())
				}
				log.Debugf("[Event] created-service %s config %s", genOBjNamespacedName(obj.Namespace, obj.Name), config.ToJson())
				m.taskCh <- NewTask(CreateTask, driver.Config{}, config)
			}
		}
	case *corev1.Node:
		log.Debugf("[Event] node %s created", obj.Name)
		m.addOrUpdateNodeIP(obj)
	}
	return handler.Result{}, nil
}

func (m *LBControlManager) addDeleteTask(svc *corev1.Service) {
	log.Debugf("[Event] service %s deleted", genOBjNamespacedName(svc.Namespace, svc.Name))
	config, err := genLBConfigByService(m.client, svc, m.clusterName, m.nodes)
	if err != nil {
		log.Warnf("[Event] deleted-service %s config gen failed %s", genOBjNamespacedName(svc.Namespace, svc.Name), err.Error())
		return
	}
	log.Debugf("[Event] deleted-service %s config %s", genOBjNamespacedName(svc.Namespace, svc.Name), config.ToJson())
	m.taskCh <- NewTask(DeleteTask, driver.Config{}, config)
}

func (m *LBControlManager) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	switch obj := e.ObjectNew.(type) {
	case *corev1.Service:
		if isServiceNeedHandle(obj) {
			if obj.ObjectMeta.DeletionTimestamp != nil {
				m.addDeleteTask(obj)
			} else {
				log.Debugf("[Event] service %s updated", genOBjNamespacedName(obj.Namespace, obj.Name))
				oldObj := e.ObjectOld.(*corev1.Service)

				old, err := genLBConfigByService(m.client, oldObj, m.clusterName, m.nodes)
				if err != nil {
					log.Warnf("[Event] updated-service %s old config gen failed %s", genOBjNamespacedName(oldObj.Namespace, oldObj.Name), err.Error())
				}

				new, err := genLBConfigByService(m.client, obj, m.clusterName, m.nodes)
				if err != nil {
					log.Warnf("[Event] updated-service %s new config gen failed %s", genOBjNamespacedName(obj.Namespace, obj.Name), err.Error())
				}
				log.Debugf("[Event] updated-service %s old config %s new config %s", genOBjNamespacedName(obj.Namespace, obj.Name), old.ToJson(), new.ToJson())
				m.taskCh <- NewTask(UpdateTask, old, new)
			}
		}
	case *corev1.Endpoints:
		log.Debugf("[Event] endpoints %s updated", genOBjNamespacedName(obj.Namespace, obj.Name))
		oldObj := e.ObjectOld.(*corev1.Endpoints)

		old, err := genLBConfigByEndpoins(m.client, oldObj, m.clusterName, m.nodes)
		if err != nil {
			log.Warnf("[Event] updated-endpoints %s old config gen failed %s", genOBjNamespacedName(oldObj.Namespace, oldObj.Name), err.Error())
		}

		new, err := genLBConfigByEndpoins(m.client, obj, m.clusterName, m.nodes)
		if err != nil {
			log.Warnf("[Event] updated-endpoints %s new config gen failed %s", genOBjNamespacedName(obj.Namespace, obj.Name), err.Error())
		}
		log.Debugf("[Event] updated-endpoints %s old config %s new config %s", genOBjNamespacedName(obj.Namespace, obj.Name), old.ToJson(), new.ToJson())
		m.taskCh <- NewTask(UpdateTask, old, new)
	}
	return handler.Result{}, nil
}

func (m *LBControlManager) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	switch obj := e.Object.(type) {
	case *corev1.Service:
		if isServiceNeedHandle(obj) {
			m.addDeleteTask(obj)
		}
	case *corev1.Node:
		log.Debugf("[Event] node %s deleted", obj.Name)
		m.deleteNodeIP(obj)
	}
	return handler.Result{}, nil
}

func (m *LBControlManager) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func (m *LBControlManager) addOrUpdateNodeIP(n *corev1.Node) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.nodes[n.Name] = n.Status.Addresses[0].Address
}

func (m *LBControlManager) deleteNodeIP(n *corev1.Node) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.nodes[n.Name]
	if !ok {
		log.Warnf("node %s doesn't exist when deleteNodeIP", n.Name)
		return
	}
	delete(m.nodes, n.Name)
}

func genOBjNamespacedName(namespace, name string) string {
	return fmt.Sprintf("{namespace:%s, name:%s}", namespace, name)
}
