package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/raintank/raintank-apps/tsdb/util"
	"github.com/raintank/worldping-api/pkg/log"
)

var (
	ElasticsearchUrl *url.URL
	IndexName        string
)

func Init(elasticsearchUrl, indexName string) error {
	var err error
	IndexName = indexName
	ElasticsearchUrl, err = url.Parse(elasticsearchUrl)
	return err
}

func Proxy(orgId int64, proxyPath string, request *http.Request) (*httputil.ReverseProxy, error) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read request body. %s", err)
	}
	searchBody, err := restrictSearch(orgId, body)
	if err != nil {
		return nil, fmt.Errorf("unable to read request body. %s", err)
	}
	log.Debug("search body is: %s", string(searchBody))

	director := func(req *http.Request) {
		req.URL.Scheme = ElasticsearchUrl.Scheme
		req.URL.Host = ElasticsearchUrl.Host
		req.URL.Path = util.JoinUrlFragments(ElasticsearchUrl.Path, proxyPath)

		req.Body = ioutil.NopCloser(bytes.NewReader(searchBody))
		req.ContentLength = int64(len(searchBody))
		req.Header.Set("Content-Length", strconv.FormatInt(req.ContentLength, 10))
	}

	proxy := &httputil.ReverseProxy{Director: director}
	return proxy, nil
}

func restrictSearch(orgId int64, body []byte) ([]byte, error) {
	var newBody bytes.Buffer

	lines := strings.Split(string(body), "\n")
	for i := 0; i < len(lines); i += 2 {
		if lines[i] == "" {
			continue
		}
		if err := validateHeader([]byte(lines[i])); err != nil {
			return newBody.Bytes(), err
		}
		newBody.Write([]byte(lines[i] + "\n"))

		s, err := transformSearch(orgId, []byte(lines[i+1]))
		if err != nil {
			return newBody.Bytes(), err
		}
		newBody.Write(s)
		newBody.Write([]byte("\n"))
	}
	return newBody.Bytes(), nil
}

type msearchHeader struct {
	SearchType        string   `json:"search_type"`
	IgnoreUnavailable bool     `json:"ignore_unavailable,omitempty"`
	Index             []string `json:"index"`
}

func validateHeader(header []byte) error {
	h := msearchHeader{}
	log.Debug("validating search header: %s", string(header))
	if err := json.Unmarshal(header, &h); err != nil {
		return err
	}
	if h.SearchType != "query_then_fetch" && h.SearchType != "count" {
		return fmt.Errorf("invalid search_type %s", h.SearchType)
	}

	for _, index := range h.Index {
		if match, err := regexp.Match("^events-\\d\\d\\d\\d-\\d\\d-\\d\\d$", []byte(index)); err != nil || !match {
			return fmt.Errorf("invalid index name. %s", index)
		}
	}

	return nil
}

type esSearch struct {
	Size            int         `json:"size"`
	Query           esQuery     `json:"query"`
	Sort            interface{} `json:"sort,omitempty"`
	Fields          interface{} `json:"fields,omitempty"`
	ScriptFields    interface{} `json:"script_fields,omitempty"`
	FielddataFields interface{} `json:"fielddata_fields,omitempty"`
	Aggs            interface{} `json:"aggs,omitempty"`
}

type esQuery struct {
	Filtered esFiltered `json:"filtered"`
}

type esFiltered struct {
	Query  interface{} `json:"query"`
	Filter esFilter    `json:"filter"`
}

type esFilter struct {
	Bool esBool `json:"bool"`
}

type esBool struct {
	Must []interface{} `json:"must"`
}

func transformSearch(orgId int64, search []byte) ([]byte, error) {
	s := esSearch{}
	if err := json.Unmarshal(search, &s); err != nil {
		return nil, err
	}

	orgCondition := map[string]map[string]int64{"term": {"org_id": orgId}}

	s.Query.Filtered.Filter.Bool.Must = append(s.Query.Filtered.Filter.Bool.Must, orgCondition)

	return json.Marshal(s)
}
