package model

import (
	"regexp"
	"time"
)

type Agent struct {
	Id      int64
	Name    string
	Enabled bool
	Owner   int64
	Public  bool
	Created time.Time
	Updated time.Time
}

type AgentTag struct {
	Id      int64
	Owner   int64
	AgentId int64
	Tag     string
	Created time.Time
}

type AgentMetric struct {
	Id       int64
	Owner    int64
	AgentId  int64
	MetricId string
}

// DTO
type AgentDTO struct {
	Id      int64     `json:"id"`
	Name    string    `json:"name"`
	Enabled bool      `json:"enabled"`
	Owner   int64     `json:"-"`
	Public  bool      `json:"public"`
	Tags    []string  `json:"tags"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

func (a *AgentDTO) ValidName() bool {
	matched, err := regexp.MatchString("^[0-9a-zA-Z_-]+$", a.Name)
	if err != nil {
		log.Errorf("regex error. %s", err)
		return false
	}

	return matched
}

type GetAgentsQuery struct {
	Name    string `json:"name"`
	Enabled string `json:"enabled"`
	Public  string `json:"public"`
	Tag     string `json:"tag"`
	Owner   int64  `json:"-"`
}
