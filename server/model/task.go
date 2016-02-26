package model

import (
	"encoding/json"
	"errors"
	"time"
)

type Task struct {
	Id       int64                             `json:"id"`
	Name     string                            `json:"name"`
	Owner    string                            `json:"owner"`
	Metrics  []string                          `json:"metrics"`
	Config   map[string]map[string]interface{} `json:"config"`
	Interval int64                             `json:"interval"`
	Route    *TaskRoute                        `json:"route"`
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
	var config interface{}

	t.Type = firstPass.Type
	switch firstPass.Type {
	case RouteAny:
	case RouteByTags:
		config = make(map[string][]string)
	case RouteByIds:
		config = make(map[string][]int64)
	default:
		return errors.New("unknown route type")
	}
	err = json.Unmarshal(firstPass.Config, &config)
	if err != nil {
		return err
	}
	t.Config = config.(map[string]interface{})
	return err
}

func (r *TaskRoute) Vaidate() (bool, error) {
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
		return false, InvalidRouteConfig
	}
	return true, nil
}

type GetTasksQuery struct {
	Metric string
	Owner  string
}
