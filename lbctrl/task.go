package lbctrl

import (
	"encoding/json"

	"github.com/zdnscloud/elb-controller/driver"

	"github.com/zdnscloud/cement/uuid"
	corev1 "k8s.io/api/core/v1"
)

type TaskType string

const (
	CreateTask TaskType = "create"
	UpdateTask TaskType = "update"
	DeleteTask TaskType = "delete"
)

type Task struct {
	ID           string          `json:"id"`
	Type         TaskType        `json:"type"`
	OldConfig    *driver.Config  `json:"oldConfig,omitempty"`
	NewConfig    *driver.Config  `json:"newConfig"`
	Failures     int             `json:"failures"`
	K8sService   *corev1.Service `json:"-"`
	ErrorMessage string          `json:"-"`
}

func NewTask(t TaskType, old, new *driver.Config, svc *corev1.Service) Task {
	id, _ := uuid.Gen()
	return Task{
		ID:         id,
		Type:       t,
		OldConfig:  old,
		NewConfig:  new,
		K8sService: svc,
	}
}

func (t Task) ToJson() string {
	b, _ := json.Marshal(&t)
	return string(b)
}
