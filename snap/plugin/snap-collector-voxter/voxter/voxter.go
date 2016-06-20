package voxter

import (
	"fmt"
	"time"

	"github.com/gosimple/slug"
	. "github.com/intelsdi-x/snap-plugin-utilities/logger"
	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
)

const (
	// Name of plugin
	Name = "voxter"
	// Version of plugin
	Version = 1
	// Type of plugin
	Type = plugin.CollectorPluginType
)

var (
	statusMap = map[string]int{"up": 0, "down": 1}
)

func init() {
	slug.CustomSub = map[string]string{".": "_"}
}

// make sure that we actually satisify requierd interface
var _ plugin.CollectorPlugin = (*Ns1)(nil)

type Ns1 struct {
}

// CollectMetrics collects metrics for testing
func (n *Ns1) CollectMetrics(mts []plugin.MetricType) ([]plugin.MetricType, error) {
	var err error
	metrics := make([]plugin.MetricType, 0)
	conf := mts[0].Config().Table()
	apiKey, ok := conf["ns1_key"]
	if !ok || apiKey.(ctypes.ConfigValueStr).Value == "" {
		LogError("ns1_key missing from config.")
		return nil, fmt.Errorf("ns1_key missing from config, %v", conf)
	}
	client, err := NewClient("https://api.nsone.net/", apiKey.(ctypes.ConfigValueStr).Value, false)
	if err != nil {
		LogError("failed to create NS1 api client.", "error", err)
		return nil, err
	}
	LogDebug("request to collect metrics", "metric_count", len(mts))
	zoneMts := make([]plugin.MetricType, 0)
	monitorMts := make([]plugin.MetricType, 0)
	for _, metricType := range mts {
		ns := metricType.Namespace()
		if len(ns) > 4 && ns[3].Value == "zones" {
			zoneMts = append(zoneMts, metricType)
		}
		if len(ns) > 4 && ns[3].Value == "monitoring" {
			monitorMts = append(monitorMts, metricType)
		}
	}

	if len(zoneMts) > 0 {
		resp, err := n.ZoneMetrics(client, zoneMts)
		if err != nil {
			LogError("failed to collect metrics.", "error", err)
			return nil, err
		}
		metrics = append(metrics, resp...)
	}
	if len(monitorMts) > 0 {
		resp, err := n.MonitorsMetrics(client, monitorMts)
		if err != nil {
			LogError("failed to collect metrics.", "error", err)
			return nil, err
		}
		metrics = append(metrics, resp...)
	}

	if err != nil {
		LogError("failed to collect metrics.", "error", err)
		return nil, err
	}
	LogDebug("collecting metrics completed", "metric_count", len(metrics))
	return metrics, nil
}

func (n *Ns1) ZoneMetrics(client *Client, mts []plugin.MetricType) ([]plugin.MetricType, error) {
	metrics := make([]plugin.MetricType, 0)
	conf := mts[0].Config().Table()
	zone, ok := conf["zone"]
	if !ok || zone.(ctypes.ConfigValueStr).Value == "" {
		LogError("zone missing from config.")
		return metrics, nil
	}
	zSlug := slug.Make(zone.(ctypes.ConfigValueStr).Value)

	qps, err := client.Qps(zone.(ctypes.ConfigValueStr).Value)
	if err != nil {
		return nil, err
	}
	metrics = append(metrics, plugin.MetricType{
		Data_:      qps.Qps,
		Namespace_: core.NewNamespace("raintank", "apps", "ns1", "zones", zSlug, "qps"),
		Timestamp_: time.Now(),
		Version_:   mts[0].Version(),
	})

	return metrics, nil
}

func (n *Ns1) MonitorsMetrics(client *Client, mts []plugin.MetricType) ([]plugin.MetricType, error) {
	metrics := make([]plugin.MetricType, 0)
	conf := mts[0].Config().Table()
	jobId, ok := conf["jobId"]
	if !ok || jobId.(ctypes.ConfigValueStr).Value == "" {
		LogError("jobId missing from config.")
		return metrics, nil
	}
	jobName, ok := conf["jobName"]
	if !ok || jobName.(ctypes.ConfigValueStr).Value == "" {
		LogError("jobName missing from config.")
		return metrics, nil
	}

	jSlug := slug.Make(jobName.(ctypes.ConfigValueStr).Value)

	j, err := client.MonitoringJobById(jobId.(ctypes.ConfigValueStr).Value)
	if err != nil {
		LogError("failed to query for job.", err)
		return nil, err
	}

	for region, status := range j.Status {
		data, ok := statusMap[status.Status]
		if !ok {
			return nil, fmt.Errorf("Unknown monitor status")
		}

		metrics = append(metrics, plugin.MetricType{
			Data_:      data,
			Namespace_: core.NewNamespace("raintank", "apps", "ns1", "monitoring", jSlug, region, "state"),
			Timestamp_: time.Now(),
			Version_:   mts[0].Version(),
		})

	}

	jobMetrics, err := client.MonitoringMetics(j.Id)
	if err != nil {
		return nil, err
	}
	for _, jm := range jobMetrics {
		for stat, m := range jm.Metrics {
			metrics = append(metrics, plugin.MetricType{
				Data_:      m.Avg,
				Namespace_: core.NewNamespace("raintank", "apps", "ns1", "monitoring", jSlug, jm.Region, stat),
				Timestamp_: time.Now(),
				Version_:   mts[0].Version(),
			})
		}
	}

	return metrics, nil
}

//GetMetricTypes returns metric types for testing
func (n *Ns1) GetMetricTypes(cfg plugin.ConfigType) ([]plugin.MetricType, error) {
	mts := []plugin.MetricType{}

	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "ns1", "zones", "*", "qps"),
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "ns1", "monitoring", "*", "*", "state"),
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "ns1", "monitoring", "*", "*", "rtt"),
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "ns1", "monitoring", "*", "*", "loss"),
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "ns1", "monitoring", "*", "*", "connect"),
		Config_:    cfg.ConfigDataNode,
	})

	return mts, nil
}

//GetConfigPolicy returns a ConfigPolicyTree for testing
func (n *Ns1) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("ns1_key", true)
	rule2, _ := cpolicy.NewStringRule("zone", false, "")
	rule3, _ := cpolicy.NewStringRule("jobId", false, "")
	rule4, _ := cpolicy.NewStringRule("jobName", false, "")
	p := cpolicy.NewPolicyNode()
	p.Add(rule)
	p.Add(rule2)
	p.Add(rule3)
	p.Add(rule4)

	c.Add([]string{"raintank", "apps", "ns1"}, p)
	return c, nil
}

//Meta returns meta data for testing
func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(
		Name,
		Version,
		Type,
		[]string{plugin.SnapGOBContentType},
		[]string{plugin.SnapGOBContentType},
		plugin.Unsecure(true),
		plugin.ConcurrencyCount(1000),
	)
}
