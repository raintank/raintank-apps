package model

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
)

type Metric struct {
	Id        string
	Namespace string
	Version   int
	Policy    []rbody.PolicyTable
	Created   time.Time
}

func (m *Metric) SetId() {
	var buffer bytes.Buffer
	buffer.WriteString(m.Namespace)
	binary.Write(&buffer, binary.LittleEndian, m.Version)
	m.Id = fmt.Sprintf("%x", md5.Sum(buffer.Bytes()))
}

type GetMetricsQuery struct {
	Namespace string `json:"namespace"`
}
