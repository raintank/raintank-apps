package model

import (
	"errors"
	"regexp"
	"time"
)

var (
	AgentNotFound = errors.New("Agent Not Found.")
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

// "url" tag is used by github.com/google/go-querystring/query
// "form" tag is used by is ued by github.com/go-macaron/binding
type GetAgentsQuery struct {
	Name    string `form:"name" url:"name,omitempty"`
	Enabled string `form:"enabled" url:"enabled,omitempty"`
	Public  string `form:"public" url:"public,omitempty"`
	Tag     string `form:"tag" url:"tag,omitempty"`
	OrderBy string `form:"orderBy" url:"orderBy,omitempty"`
	Limit   int    `form:"limit" url:"limit,omitempty"`
	Page    int    `form:"page" url:"page,omitempty"`
	Owner   int64  `form:"-" url:"-"`
}
