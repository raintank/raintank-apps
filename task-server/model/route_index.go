package model

import (
	"time"
)

type RouteByIdIndex struct {
	Id      int64
	TaskId  int64
	AgentId int64
	Created time.Time
}

type RouteByTagIndex struct {
	Id      int64
	TaskId  int64
	OrgId   int64
	Tag     string
	Created time.Time
}

type RouteByAnyIndex struct {
	Id      int64
	TaskId  int64
	AgentId int64
	Created time.Time
}
