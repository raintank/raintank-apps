package model

import (
	"errors"
	"time"

	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
)

var (
	MetricAlreadyExists = errors.New("Metric already exists.")
)

type Metric struct {
	Id        int64               `json:"-"`
	Owner     int64               `json:"-"`
	Public    bool                `json:"public"`
	Namespace string              `json:"namespace"`
	Version   int64               `json:"version"`
	Policy    []rbody.PolicyTable `json:"policy"`
	Created   time.Time           `json:"created"`
}

// "url" tag is used by github.com/google/go-querystring/query
// "form" tag is used by is ued by github.com/go-macaron/binding
type GetMetricsQuery struct {
	Namespace string `form:"namespace" url:"namespace,omitempty"`
	Version   int64  `form:"version" url:"version, omitempty"`
	Owner     int64  `form:"-" url:"-"`
	OrderBy   string `form:"orderBy" url:"orderBy,omitempty"`
	Limit     int    `form:"limit" url:"limit,omitempty"`
	Page      int    `form:"page" url:"page,omitempty"`
}
