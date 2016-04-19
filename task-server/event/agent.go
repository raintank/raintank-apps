package event

import (
	"encoding/json"
	"time"

	"github.com/raintank/raintank-apps/task-server/model"
)

type AgentCreated struct {
	Ts      time.Time
	Payload *model.AgentDTO
}

func (a *AgentCreated) Type() string {
	return "agent.created"
}

func (a *AgentCreated) Timestamp() time.Time {
	return a.Ts
}

func (a *AgentCreated) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}

type AgentDeleted struct {
	Ts      time.Time
	Payload *model.AgentDTO
}

func (a *AgentDeleted) Type() string {
	return "agent.deleted"
}

func (a *AgentDeleted) Timestamp() time.Time {
	return a.Ts
}

func (a *AgentDeleted) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}

type AgentUpdated struct {
	Ts      time.Time
	Payload struct {
		Old *model.AgentDTO `json:"old"`
		New *model.AgentDTO `json:"new"`
	}
}

func (a *AgentUpdated) Type() string {
	return "agent.updated"
}

func (a *AgentUpdated) Timestamp() time.Time {
	return a.Ts
}

func (a *AgentUpdated) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}

type AgentOnline struct {
	Ts      time.Time
	Payload *model.AgentDTO
}

func (a *AgentOnline) Type() string {
	return "agent.online"
}

func (a *AgentOnline) Timestamp() time.Time {
	return a.Ts
}

func (a *AgentOnline) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}

type AgentOffline struct {
	Ts      time.Time
	Payload *model.AgentDTO
}

func (a *AgentOffline) Type() string {
	return "agent.offline"
}

func (a *AgentOffline) Timestamp() time.Time {
	return a.Ts
}

func (a *AgentOffline) Body() ([]byte, error) {
	return json.Marshal(a.Payload)
}
