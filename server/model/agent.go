package model

import (
	"regexp"
	"strings"
	"time"
)

type Agent struct {
	Id       int64
	Name     string
	Slug     string
	Password string
	Enabled  bool
	Owner    string
	Public   bool
	Created  time.Time
	Updated  time.Time
}

func (agent *Agent) UpdateSlug() {
	name := strings.ToLower(agent.Name)
	re := regexp.MustCompile("[^\\w ]+")
	re2 := regexp.MustCompile("\\s")
	agent.Slug = re2.ReplaceAllString(re.ReplaceAllString(name, ""), "-")
}

type AgentTag struct {
	Id      int64
	Owner   string
	AgentId int64
	Tag     string
	Created time.Time
}

type AgentMetric struct {
	Id       int64
	Owner    string
	AgentId  int64
	MetricId string
}

// DTO
type AgentDTO struct {
	Id       int64     `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Password string    `json:"password"`
	Enabled  bool      `json:"enabled"`
	Owner    string    `json:"-"`
	Public   bool      `json:"public"`
	Tags     []string  `json:"tags"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

type GetAgentsQuery struct {
	Name    string `json:"name"`
	Enabled string `json:"enabled"`
	Public  string `json:"public"`
	Tag     string `json:"tag"`
	Owner   string `json:"-"`
}
