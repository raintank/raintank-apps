package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Typed errors
var (
	ErrEndpointNotFound = errors.New("Endpoint not found")
)

type Endpoint struct {
	Id      int64
	Owner   int64
	Name    string
	Slug    string
	Created time.Time
	Updated time.Time
}

func (endpoint *Endpoint) UpdateSlug() {
	name := strings.ToLower(endpoint.Name)
	re := regexp.MustCompile("[^\\w ]+")
	re2 := regexp.MustCompile("\\s")
	endpoint.Slug = re2.ReplaceAllString(re.ReplaceAllString(name, "_"), "-")
}

type EndpointTag struct {
	Id         int64
	Owner      int64
	EndpointId int64
	Tag        string
	Created    time.Time
}

// ---------------
// DTOs
type EndpointDTO struct {
	Id      int64     `json:"id"`
	Owner   int64     `json:"owner"`
	Name    string    `json:"name" binding:"Required"`
	Slug    string    `json:"slug"`
	Checks  []*Check  `json:"checks"`
	Tags    []string  `json:"tags"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

type CheckType string

const (
	HTTP_CHECK  CheckType = "http"
	HTTPS_CHECK CheckType = "https"
	DNS_CHECK   CheckType = "dns"
	PING_CHECK  CheckType = "ping"
)

type Check struct {
	Id             int64                  `json:"id"`
	Owner          int64                  `json:"-"`
	EndpointId     int64                  `json:"-"`
	Type           CheckType              `json:"type" binding:"Required,In(http,https,dns,ping)"`
	Frequency      int64                  `json:"frequency" binding:"Required,Range(10,300)"`
	Enabled        bool                   `json:"enabled"`
	State          CheckEvalResult        `json:"state"`
	StateChange    time.Time              `json:"stateChange"`
	StateCheck     time.Time              `json:"stateCheck"`
	Settings       map[string]interface{} `json:"settings" binding:"Required"`
	HealthSettings *CheckHealthSettings   `json:"healthSettings"`
	Created        time.Time              `json:"created"`
	Updated        time.Time              `json:"updated"`
	TaskId         int64                  `json:"-"`
}

func (c *Check) IsValid() bool {
	if !(c.Type == HTTP_CHECK || c.Type == HTTPS_CHECK || c.Type == DNS_CHECK || c.Type == PING_CHECK) {
		return false
	}

	//TODO check settings
	return true
}

type CheckEvalResult int

const (
	EvalResultOK CheckEvalResult = iota
	EvalResultWarn
	EvalResultCrit
	EvalResultUnknown = -1
)

func (c CheckEvalResult) String() string {
	switch c {
	case EvalResultOK:
		return "OK"
	case EvalResultWarn:
		return "Warning"
	case EvalResultCrit:
		return "Critical"
	case EvalResultUnknown:
		return "Unknown"
	default:
		panic(fmt.Sprintf("Invalid CheckEvalResult value %d", int(c)))
	}
}

type CheckHealthSettings struct {
	NumCollectors int                      `json:"numCollectors" binding:"Required,Range(1,20)"`
	Steps         int                      `json:"steps" binding:"Required,Range(1,5)"`
	Notifications CheckNotificationSetting `json:"notifications"`
}

type CheckNotificationSetting struct {
	Enabled   bool   `json:"enabled"`
	Addresses string `json:"addresses"`
}

// implement the go-xorm/core.Conversion interface
func (e *CheckHealthSettings) FromDB(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *CheckHealthSettings) ToDB() ([]byte, error) {
	return json.Marshal(e)
}

type GetEndpointsQuery struct {
	Owner   int64
	Name    string
	Tag     string
	OrderBy string `binding:"In(name,slug,created,updated,)"`
	Limit   int    `binding:"Range(0,100)"`
	Page    int
}

type DiscoverEndpointCmd struct {
	Name string `form:"name"`
}
