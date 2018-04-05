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
	"time"

	"github.com/grafana/metrictank/stats"

	"github.com/google/go-querystring/query"
)

// APIVersion NS1
const APIVersion = "v1"

var (
	ns1ClientQueries      = stats.NewCounter64("collector.ns1.client.queries.count")
	ns1ClientAuthFailures = stats.NewCounter64("collector.ns1.client.authfailures.count")
)

var (
	ErrNotFound     = errors.New("Not Found")
	ErrAuthFailure  = errors.New("Authentication failed")
	ErrAccessDenied = errors.New("Access denied")
	ErrNilResponse  = errors.New("Nil response")
)

// Zone stores zone name
type Zone struct {
	Id   string `json:"id"`
	Zone string `json:"zone"`
}

// QPS Queries Per Second
type QPS struct {
	QPS float64 `json:"qps"`
}

// Client holds configuration for the connection
type Client struct {
	URL    *url.URL
	http   *http.Client
	APIKey string
	prefix string
}

// NewClient creates a new client to pull data from NS1 API
func NewClient(serverURL, apiKey string, insecure bool) (*Client, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL %s is not in the format of http(s)://<ip>:<port>", serverURL)
	}
	u.Path = path.Clean(u.Path + "/" + APIVersion)
	c := &Client{
		URL:    u,
		APIKey: apiKey,
		http: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecure,
				},
			},
			Timeout: time.Second * 60,
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
	req.Header.Set("X-NSONE-KEY", c.APIKey)
	rsp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	return handleResp(rsp)
}

func handleResp(rsp *http.Response) ([]byte, error) {
	b, err := ioutil.ReadAll(rsp.Body)
	rsp.Body.Close()
	ns1ClientQueries.Inc()
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode == 401 {
		ns1ClientAuthFailures.Inc()
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

	return b, nil
}

// ToQueryString Convert an interface{} to a urlencoded querystring
func ToQueryString(q interface{}) (string, error) {
	v, err := query.Values(q)
	if err != nil {
		return "", err
	}
	return v.Encode(), nil
}

// Zones gets the zones and decides into array
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

// QPS gets the qps metric from NS1 API
func (c *Client) QPS(zone string) (*QPS, error) {
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
	qps := QPS{}
	err = json.Unmarshal(body, &qps)
	if err != nil {
		return nil, err
	}
	return &qps, nil
}
