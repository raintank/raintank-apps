package main

import (
	"net/url"

	"github.com/intelsdi-x/snap/mgmt/rest/client"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
)

var SnapClient *client.Client

func InitSnapClient(u *url.URL) {
	SnapClient = client.New(u.String(), "v1", false)
}

func GetSnapMetrics() ([]*rbody.Metric, error) {
	resp := SnapClient.GetMetricCatalog()
	return resp.Catalog, resp.Err
}
