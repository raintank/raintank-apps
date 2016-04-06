package model

import (
	"time"
)

type ProbeDTO struct {
	Id            int64     `json:"id"`
	OrgId         int64     `json:"orgId"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	Tags          []string  `json:"tags"`
	Public        bool      `json:"public"`
	Enabled       bool      `json:"enabled"`
	EnabledChange time.Time `json:"enabledChange"`
	Online        bool      `json:"online"`
	OnlineChange  time.Time `json:"onlineChange"`
}

type GetProbesQuery struct {
	Name    string `form:"name" url:"name,omitempty"`
	Enabled string `form:"enabled" url:"enabled,omitempty"`
	Public  string `form:"public" url:"public,omitempty"`
	Tag     string `form:"tag" url:"tag,omitempty"`
	OrderBy string `form:"orderBy" url:"orderBy,omitempty"`
	Limit   int    `form:"limit" url:"limit,omitempty"`
	Page    int    `form:"page" url:"page,omitempty"`
	OrgId   int64  `form:"-" url:"-"`
}
