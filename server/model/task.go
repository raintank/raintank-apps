package model

import (
	"encoding/json"
	"errors"
	"time"
)

type Task struct {
	Id       int64
	Name     string
	Owner    string
	Config   map[string]map[string]interface{}
	Interval int64
	Route    *TaskRoute
	Enabled  bool
	Created  time.Time
	Updated  time.Time
}

type TaskMetric struct {
	Id        int64
	TaskId    int64
	Namespace string
	Version   int64
	Created   time.Time
}

type TaskDTO struct {
	Id       int64                             `json:"id"`
	Name     string                            `json:"name"`
	Owner    string                            `json:"-"`
	Config   map[string]map[string]interface{} `json:"config"`
	Interval int64                             `json:"interval"`
	Route    *TaskRoute                        `json:"route"`
	Metrics  map[string]int64                  `json:"metrics"`
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
	InvalidRouteConfig = errors.New("Invlid route config")
)

type TaskRoute struct {
	Type   RouteType              `json:"type"`
	Config map[string]interface{} `json:"config"`
}

func (t *TaskRoute) ToDB() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TaskRoute) FromDB(data []byte) error {
	return json.Unmarshal(data, t)
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
		c := make(map[string]int64)
		err = json.Unmarshal(firstPass.Config, &c)
		if err != nil {
			return err
		}
		for k, v := range c {
			config[k] = v
		}
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
		return errors.New("unknown route type")
	}

	t.Config = config
	return err
}

func (r *TaskRoute) Validate() (bool, error) {
	switch r.Type {
	case RouteAny:
		if len(r.Config) != 1 {
			return false, InvalidRouteConfig
		}
		if _, ok := r.Config["id"]; !ok {
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
		return false, InvalidRouteConfig
	}
	return true, nil
}

type GetTasksQuery struct {
	Metric        string `json:"metric"`
	MetricVersion int64  `json:"metric_version"`
	Owner         string `json:"-"`
	Enabled       string `json:"enabled"`
}
