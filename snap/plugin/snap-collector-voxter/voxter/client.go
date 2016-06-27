package voxter

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

var (
	ErrNotFound     = errors.New("Not Found")
	ErrAuthFailure  = errors.New("Authentication failed")
	ErrAccessDenied = errors.New("Access denied")
	ErrNilResponse  = errors.New("Nil response")
)

type Client struct {
	URL    *url.URL
	http   *http.Client
	ApiKey string
	prefix string
}

type VoxterRet struct {
	Success bool `json:"success"`
	Data *VoxterData `json:"data"`
}

type VoxterData struct {
	Network map[string]string `json:"network"`
	Counters map[string]*Endpoint `json:"counters"`
}

type VoxterChannels struct {
	Inbound float64 `json:"inbound"`
	Outbound float64 `json:"outbound"`
}

type Endpoint struct {
	Channels *VoxterChannels `json:"channels"`
	Registrations float64 `json:"registrations"`
}

func NewClient(serverUrl, apiKey string, insecure bool) (*Client, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL %s is not in the format of http(s)://<ip>:<port>", serverUrl)
	}
	u.Path = path.Clean(u.Path)
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
	req.Header.Set("X-API-KEY", c.ApiKey)
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

func (c *Client) EndpointStats() (map[string]*Endpoint, error) {
	body, err := c.get("/stats/piston", nil)
	if err != nil {
		return nil, err
	}

	ret := new(VoxterRet)
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}

	return ret.Data.Counters, nil
}

