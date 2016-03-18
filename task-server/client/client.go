package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/google/go-querystring/query"
	"github.com/raintank/raintank-apps/task-server/api/rbody"
)

const Version = "v1"

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

func New(serverUrl, apiKey string, insecure bool) (*Client, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL %s is not in the format of http(s)://<ip>:<port>", serverUrl)
	}
	u.Path = path.Clean(u.Path + "/api/" + Version)
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

func (c *Client) get(path string, query interface{}) (*rbody.ApiResponse, error) {
	if query != nil {
		qstr, err := ToQueryString(query)
		if err != nil {
			return nil, err
		}
		path = path + "?" + qstr
	}
	return c.do("GET", path, nil)
}

func (c *Client) put(path string, body interface{}) (*rbody.ApiResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do("PUT", path, b)
}

func (c *Client) post(path string, body interface{}) (*rbody.ApiResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do("POST", path, b)
}

func (c *Client) delete(path string, body interface{}) (*rbody.ApiResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do("DELETE", path, b)
}

func (c *Client) Heartbeat() (bool, error) {
	resp, err := c.do("GET", "/", nil)
	if err != nil {
		return false, err
	}
	if err := resp.Error(); err != nil {
		return false, err
	}
	if resp.Meta.Type != "heartbeat" {
		return false, fmt.Errorf("invalid responseMeta. Expected type: heartbeat, got %s", resp.Meta.Type)
	}
	return true, nil
}

func (c *Client) do(method, path string, body []byte) (*rbody.ApiResponse, error) {
	var (
		rsp *http.Response
		err error
		req *http.Request
	)
	switch method {
	case "GET":
		req, err = http.NewRequest(method, c.prefix+path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		rsp, err = c.http.Do(req)
		if err != nil {
			return nil, err
		}
	case "PUT":
		var b *bytes.Reader
		b = bytes.NewReader(body)

		req, err = http.NewRequest(method, c.prefix+path, b)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		req.Header.Add("Content-Type", "application/json")

		rsp, err = c.http.Do(req)
		if err != nil {
			return nil, err
		}
	case "DELETE":
		var b *bytes.Reader
		b = bytes.NewReader(body)

		req, err = http.NewRequest(method, c.prefix+path, b)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		req.Header.Add("Content-Type", "application/json")
		rsp, err = c.http.Do(req)
		if err != nil {
			return nil, err
		}
	case "POST":
		var b *bytes.Reader
		b = bytes.NewReader(body)

		req, err = http.NewRequest(method, c.prefix+path, b)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.ApiKey)
		req.Header.Add("Content-Type", "application/json")
		rsp, err = c.http.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return handleResp(rsp)
}

func handleResp(rsp *http.Response) (*rbody.ApiResponse, error) {
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
	resp := new(rbody.ApiResponse)
	err = json.Unmarshal(b, resp)
	// If unmarshaling fails show first part of response to help debug
	// connection issues.
	if err != nil {
		limit := 1000
		if len(b) > limit {
			limit = len(b)
		}
		return nil, fmt.Errorf("Unknown API response: %s\n\n Received: %s", err, string(b[:limit]))
	}
	if resp == nil {
		// Catch corner case where JSON gives no error but resp is nil
		return nil, ErrNilResponse
	}
	return resp, nil
}

// Convert an interface{} to a urlencoded querystring
func ToQueryString(q interface{}) (string, error) {
	v, err := query.Values(q)
	if err != nil {
		return "", err
	}
	return v.Encode(), nil
}
