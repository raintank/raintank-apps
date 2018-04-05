package model

import (
	"errors"
	"regexp"
	"time"

	"github.com/raintank/worldping-api/pkg/log"
)

var (
	AgentNotFound = errors.New("Agent Not Found.")
)

type Agent struct {
	Id            int64
	Name          string
	Enabled       bool
	EnabledChange time.Time
	OrgId         int64
	Public        bool
	Online        bool
	OnlineChange  time.Time
	Created       time.Time
	Updated       time.Time
}

type AgentTag struct {
	Id      int64
	OrgId   int64
	AgentId int64
	Tag     string
	Created time.Time
}

// DTO
type AgentDTO struct {
	Id            int64     `json:"id"`
	Name          string    `json:"name" binding:"Required"`
	Enabled       bool      `json:"enabled"`
	EnabledChange time.Time `json:"enabledChange"`
	OrgId         int64     `json:"-"`
	Public        bool      `json:"public"`
	Tags          []string  `json:"tags"`
	Online        bool      `json:"online"`
	OnlineChange  time.Time `json:"onlineChange"`
	Created       time.Time `json:"created"`
	Updated       time.Time `json:"updated"`
}

func (a *AgentDTO) ValidName() bool {
	matched, err := regexp.MatchString("^[0-9a-zA-Z_-]+$", a.Name)
	if err != nil {
		log.Error(3, "regex error. %s", err)
		return false
	}

	return matched
}

// "url" tag is used by github.com/google/go-querystring/query
// "form" tag is used by is ued by github.com/go-macaron/binding
type GetAgentsQuery struct {
	Name    string   `form:"name" url:"name,omitempty"`
	Enabled string   `form:"enabled" url:"enabled,omitempty"`
	Public  string   `form:"public" url:"public,omitempty"`
	Tag     []string `form:"tag" url:"tag,omitempty"`
	OrderBy string   `form:"orderBy" url:"orderBy,omitempty"`
	Limit   int      `form:"limit" url:"limit,omitempty"`
	Page    int      `form:"page" url:"page,omitempty"`
	OrgId   int64    `form:"-" url:"-"`
}
