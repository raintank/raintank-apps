package event

import (
	"encoding/json"
	"time"

	"github.com/raintank/raintank-apps/task-server/model"
)

type TaskCreated struct {
	Ts      time.Time
	Payload *model.TaskDTO
}

func (a *TaskCreated) Type() string {
	return "task.created"
}

func (a *TaskCreated) Timestamp() time.Time {
	return a.Ts
}

func (a *TaskCreated) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}

type TaskDeleted struct {
	Ts      time.Time
	Payload *model.TaskDTO
}

func (a *TaskDeleted) Type() string {
	return "task.deleted"
}

func (a *TaskDeleted) Timestamp() time.Time {
	return a.Ts
}

func (a *TaskDeleted) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}

type TaskUpdated struct {
	Ts      time.Time
	Payload struct {
		Last    *model.TaskDTO `json:"old"`
		Current *model.TaskDTO `json:"new"`
	}
}

func (a *TaskUpdated) Type() string {
	return "task.updated"
}

func (a *TaskUpdated) Timestamp() time.Time {
	return a.Ts
}

func (a *TaskUpdated) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}
