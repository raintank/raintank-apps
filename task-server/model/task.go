package model

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	TaskNotFound = errors.New("Task Not Found.")
)

type Task struct {
	Id       int64
	Name     string
	TaskType string
	OrgId    int64
	Config   map[string]map[string]interface{}
	Interval int64
	Route    *TaskRoute `xorm:"JSON"`
	Enabled  bool
	Created  time.Time
	Updated  time.Time
}

type TaskDTO struct {
	Id       int64                             `json:"id"`
	Name     string                            `json:"name" binding:"Required"`
	TaskType string                            `json:"taskType"`
	OrgId    int64                             `json:"orgId"`
	Config   map[string]map[string]interface{} `json:"config"`
	Interval int64                             `json:"interval" binding:"Required"`
	Route    *TaskRoute                        `xorm:"JSON" json:"route" binding:"Required"`
	Enabled  bool                              `json:"enabled"`
	Created  time.Time                         `json:"created"`
	Updated  time.Time                         `json:"updated"`
}

type RouteType string

const (
	RouteAny    RouteType = "any"
	RouteByTags RouteType = "byTags"
	RouteByIds  RouteType = "byIds"
)

var (
	InvalidRouteConfig = errors.New("Invalid route config")
	UnknownRouteType   = errors.New("unknown route type")
)

type TaskRoute struct {
	Type   RouteType              `json:"type" binding:"Required"`
	Config map[string]interface{} `json:"config"`
}

func (t *TaskRoute) UnmarshalJSON(body []byte) error {
	type delay struct {
		Type   RouteType       `json:"type"`
		Config json.RawMessage `json:"config"`
	}
	firstPass := delay{}
	err := json.Unmarshal(body, &firstPass)

	if err != nil {
		return err
	}
	config := make(map[string]interface{})

	t.Type = firstPass.Type
	switch firstPass.Type {
	case RouteAny:
		//do nothing.
	case RouteByTags:
		c := make(map[string][]string)
		err = json.Unmarshal(firstPass.Config, &c)
		if err != nil {
			return err
		}
		for k, v := range c {
			config[k] = v
		}
	case RouteByIds:
		c := make(map[string][]int64)
		err = json.Unmarshal(firstPass.Config, &c)
		if err != nil {
			return err
		}
		for k, v := range c {
			config[k] = v
		}
	default:
		return UnknownRouteType
	}

	t.Config = config
	return err
}

func (r *TaskRoute) Validate() (bool, error) {
	switch r.Type {
	case RouteAny:
		if len(r.Config) != 0 {
			return false, InvalidRouteConfig
		}
	case RouteByTags:
		if len(r.Config) != 1 {
			return false, InvalidRouteConfig
		}
		if _, ok := r.Config["tags"]; !ok {
			return false, InvalidRouteConfig
		}
	case RouteByIds:
		if len(r.Config) != 1 {
			return false, InvalidRouteConfig
		}
		if _, ok := r.Config["ids"]; !ok {
			return false, InvalidRouteConfig
		}
	default:
		return false, UnknownRouteType
	}
	return true, nil
}

// "url" tag is used by github.com/google/go-querystring/query
// "form" tag is used by is ued by github.com/go-macaron/binding
type GetTasksQuery struct {
	Name     string `form:"name" url:"name,omitempty"`
	Metric   string `form:"metric" url:"metric,omitempty"`
	TaskType string `form:"taskType" url:"taskType,omitempty"`
	OrgId    int64  `form:"-" url:"-"`
	Enabled  string `form:"enabled" url:"enabled,omitempty"`
	OrderBy  string `form:"orderBy" url:"orderBy,omitempty"`
	Limit    int    `form:"limit" url:"limit,omitempty"`
	Page     int    `form:"page" url:"page,omitempty"`
}
