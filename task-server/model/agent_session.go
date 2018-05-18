package model

import (
	"time"
)

type AgentSession struct {
	Id        string
	AgentId   int64
	Version   int64
	RemoteIp  string
	Server    string
	Created   time.Time
	Heartbeat time.Time
}
