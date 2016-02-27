package model

type RouteByIdIndex struct {
	Id      int64
	TaskId  int64
	AgentId int64
}

type RouteByTagIndex struct {
	Id     int64
	TaskId int64
	Tag    string
}
