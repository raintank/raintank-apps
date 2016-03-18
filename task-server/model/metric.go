package model

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"time"

	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
)

var (
	MetricAlreadyExists = errors.New("Metric already exists.")
)

type Metric struct {
	Id        string
	Owner     int64
	Public    bool
	Namespace string
	Version   int64
	Policy    []rbody.PolicyTable
	Created   time.Time
}

func (m *Metric) SetId() {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%t:%d:%s:%d", m.Public, m.Owner, m.Namespace, m.Version))
	m.Id = fmt.Sprintf("%x", md5.Sum(buffer.Bytes()))
}

// "url" tag is used by github.com/google/go-querystring/query
// "form" tag is used by is ued by github.com/go-macaron/binding
type GetMetricsQuery struct {
	Namespace string `form:"namespace" url:"namespace,omitempty"`
	Version   int64  `form:"version" url:"version, omitempty"`
	Owner     int64  `form:"-" url:"-"`
}
