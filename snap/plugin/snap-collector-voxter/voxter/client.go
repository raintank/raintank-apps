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

type VoxChannels struct {
	Inbound int
	Outbound int
}

type Endpoint struct {
	Name string
	Channels VoxChannels
	Registrations int
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

func (c *Client) EndpointStats() ([]*Endpoint, error) {
	body, err := c.get("/stats/piston", nil)
	if err != nil {
		return nil, err
	}

	raw := make(map[string]interface{})
	err = json.Unmarshal(body, &raw)
	if err != nil {
		return nil, err
	}
	endpoints := make([]*Endpoint, 0, len(raw))
	for k, v := range raw {
		e := new(Endpoint)
		e.Name = k
		e.Registrations = v.(map[string]interface{})["registrations"].(int)
		e.Channels.Inbound = v.(map[string]interface{})["channels"].(map[string]int)["inbound"]
		e.Channels.Outbound = v.(map[string]interface{})["channels"].(map[string]int)["outbound"]
		endpoints = append(endpoints, e)
	}
	
	return endpoints, nil
}

