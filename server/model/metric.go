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
	Owner     string
	Public    bool
	Namespace string
	Version   int64
	Policy    []rbody.PolicyTable
	Created   time.Time
}

func (m *Metric) SetId() {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("%t:%s:%s:%d", m.Public, m.Owner, m.Namespace, m.Version))
	m.Id = fmt.Sprintf("%x", md5.Sum(buffer.Bytes()))
}

type GetMetricsQuery struct {
	Namespace string `json:"namespace"`
	Version   int64  `json:"version"`
	Owner     string `json:"-"`
}
