package lbctrl

import (
	"encoding/json"

	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/elb-controller/driver"
)

type TaskType string

const (
	CreateTask TaskType = "create"
	UpdateTask TaskType = "update"
	DeleteTask TaskType = "delete"
)

type Task struct {
	ID        string        `json:"id"`
	Type      TaskType      `json:"type"`
	OldConfig driver.Config `json:"oldConfig,omitempty"`
	NewConfig driver.Config `json:"newConfig"`
	Failures  int           `json:"failures"`
}

func NewTask(t TaskType, old, new driver.Config) Task {
	id, _ := uuid.Gen()
	return Task{
		ID:        id,
		Type:      t,
		OldConfig: old,
		NewConfig: new,
	}
}

func (t Task) ToJson() string {
	b, _ := json.Marshal(&t)
	return string(b)
}
