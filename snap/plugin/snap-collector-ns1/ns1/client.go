package ns1

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/google/go-querystring/query"
)

const ApiVersion = "v1"

var (
	ErrNotFound     = errors.New("Not Found")
	ErrAuthFailure  = errors.New("Authentication failed")
	ErrAccessDenied = errors.New("Access denied")
	ErrNilResponse  = errors.New("Nil response")
)

type Zone struct {
	Id   string `json:"id"`
	Zone string `json:"zone"`
}

type Qps struct {
	Qps float64 `json:"qps"`
}

type MonitoringJob struct {
	Id        string                `json:"id"`
	Name      string                `json:"name"`
	Status    map[string]*JobStatus `json:"status"`
	Frequency json.Number           `json:"frequency"`
}

type JobStatus struct {
	Since  json.Number `json:"since"`
	Status string      `json:"status"`
}

type MonitoringMetric struct {
	JobId   string             `json:"jobid"`
	Region  string             `json:"region"`
	Metrics map[string]*Metric `json:"metrics"`
}

type Metric struct {
	Avg   float64  `json:"avg"`
	Graph []*Point `json:"graph"`
}

type Point struct {
	Timestamp int
	Value     float64
}

func (p *Point) UnmarshalJSON(data []byte) error {
	tmp := make([]json.Number, 0)
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	if len(tmp) != 2 {
		return fmt.Errorf("data array should only have 2 values, [ts,val]")
	}
	ts, err := tmp[0].Int64()
	if err != nil {
		return err
	}
	p.Timestamp = int(ts)
	p.Value, err = tmp[1].Float64()
	if err != nil {
		return err
	}
	return nil
}

type Client struct {
	URL    *url.URL
	http   *http.Client
	ApiKey string
	prefix string
}

func NewClient(serverUrl, apiKey string, insecure bool) (*Client, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL %s is not in the format of http(s)://<ip>:<port>", serverUrl)
	}
	u.Path = path.Clean(u.Path + "/" + ApiVersion)
	c := &Client{
		URL:    u,
		ApiKey: apiKey,
		http: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecure,
				},
			},
		},
		prefix: u.String(),
	}
	return c, nil
}

func (c *Client) get(path string, query interface{}) ([]byte, error) {
	if query != nil {
		qstr, err := ToQueryString(query)
		if err != nil {
			return nil, err
		}
		path = path + "?" + qstr
	}
	log.Printf("sending request for %s", c.prefix+path)
	req, err := http.NewRequest("GET", c.prefix+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-NSONE-KEY", c.ApiKey)
	rsp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	return handleResp(rsp)
}

func handleResp(rsp *http.Response) ([]byte, error) {
	if rsp.StatusCode == 401 {
		return nil, ErrAuthFailure
	}
	if rsp.StatusCode == 403 {
		return nil, ErrAccessDenied
	}
	if rsp.StatusCode == 404 {
		return nil, ErrNotFound
	}
	if rsp.StatusCode != 200 {
		return nil, fmt.Errorf("Unknown error encountered. %s", rsp.Status)
	}
	b, err := ioutil.ReadAll(rsp.Body)
	rsp.Body.Close()
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Convert an interface{} to a urlencoded querystring
func ToQueryString(q interface{}) (string, error) {
	v, err := query.Values(q)
	if err != nil {
		return "", err
	}
	return v.Encode(), nil
}

func (c *Client) Zones() ([]*Zone, error) {
	body, err := c.get("/zones", nil)
	if err != nil {
		return nil, err
	}
	zones := make([]*Zone, 0)
	err = json.Unmarshal(body, &zones)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (c *Client) Qps(zone string) (*Qps, error) {
	path := "/stats/qps"
	if zone != "" {
		// we need to escape twice as internally the path is stored in encoded
		// form so it is not possible to tell if %2F or / were passed.
		// see https://golang.org/pkg/net/url/#URL
		path = path + "/" + url.QueryEscape(url.QueryEscape(zone))
	}
	body, err := c.get(path, nil)
	if err != nil {
		log.Printf("failed to get %s. %s", path, err)
		return nil, err
	}
	qps := Qps{}
	err = json.Unmarshal(body, &qps)
	if err != nil {
		return nil, err
	}
	return &qps, nil
}

func (c *Client) MonitoringJobs() ([]*MonitoringJob, error) {
	body, err := c.get("/monitoring/jobs", nil)
	if err != nil {
		return nil, err
	}
	jobs := make([]*MonitoringJob, 0)
	err = json.Unmarshal(body, &jobs)
	if err != nil {
		log.Printf("failed to unmarshal monitoringJob resp. %s", err)
		log.Printf("--------\n%s\n--------\n", body)
		return nil, err
	}
	return jobs, nil
}

func (c *Client) MonitoringMetics(jobid string) ([]*MonitoringMetric, error) {
	body, err := c.get("/monitoring/metrics/"+jobid, nil)
	if err != nil {
		return nil, err
	}
	metrics := make([]*MonitoringMetric, 0)
	err = json.Unmarshal(body, &metrics)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}
