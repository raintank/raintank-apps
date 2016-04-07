package ns1

import (
	"fmt"
	"os"
	"time"

	"github.com/gosimple/slug"
	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
)

const (
	// Name of plugin
	Name = "ns1"
	// Version of plugin
	Version = 1
	// Type of plugin
	Type = plugin.CollectorPluginType
)

var (
	statusMap = map[string]int{"up": 0, "down": 1}
	hostname  = ""
)

func init() {
	hostname, _ = os.Hostname()
}

// make sure that we actually satisify requierd interface
var _ plugin.CollectorPlugin = (*Ns1)(nil)

type Ns1 struct {
}

// CollectMetrics collects metrics for testing
func (n *Ns1) CollectMetrics(mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	var err error
	metrics := make([]plugin.PluginMetricType, 0)
	conf := mts[0].Config().Table()
	apiKey, ok := conf["ns1_key"]
	if !ok || apiKey.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("ns1_key missing from config, %v", conf)
	}
	client, err := NewClient("https://api.nsone.net/", apiKey.(ctypes.ConfigValueStr).Value, false)
	if err != nil {
		return nil, err
	}
	for _, metricType := range mts {
		ns := metricType.Namespace()
		if ns[3] == "_all" {
			m, err := n.AllMetrics(client, metricType)
			return m, err
		}
		// TODO, scan through the metrics and put them into zone and monitor metrics.
	}
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (n *Ns1) AllMetrics(client *Client, mt plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	metrics := make([]plugin.PluginMetricType, 0)
	zm, err := n.ZoneMetrics(client, []plugin.PluginMetricType{mt})
	if err != nil {
		return nil, err
	}
	metrics = append(metrics, zm...)
	mm, err := n.MonitorsMetrics(client, []plugin.PluginMetricType{mt})
	if err != nil {
		return nil, err
	}
	metrics = append(metrics, mm...)
	return metrics, nil
}

func (n *Ns1) ZoneMetrics(client *Client, mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	metrics := make([]plugin.PluginMetricType, 0)
	zones, err := client.Zones()
	if err != nil {
		return nil, err
	}

	for _, z := range zones {
		zSlug := slug.Make(z.Zone)
		qps, err := client.Qps(z.Zone)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, plugin.PluginMetricType{
			Data_:      qps.Qps,
			Namespace_: []string{"raintank", "apps", "ns1", "zones", zSlug, "qps"},
			Source_:    hostname,
			Timestamp_: time.Now(),
			Labels_:    []core.Label{{Index: 4, Name: "zone"}},
			Version_:   mts[0].Version(),
		})
	}
	return metrics, nil
}

func (n *Ns1) MonitorsMetrics(client *Client, mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	metrics := make([]plugin.PluginMetricType, 0)
	jobs, err := client.MonitoringJobs()
	if err != nil {
		return nil, err
	}
	for _, j := range jobs {
		jSlug := slug.Make(j.Name)
		for region, status := range j.Status {
			data, ok := statusMap[status.Status]
			if !ok {
				return nil, fmt.Errorf("Unknown monitor status")
			}
			metrics = append(metrics, plugin.PluginMetricType{
				Data_:      data,
				Namespace_: []string{"raintank", "apps", "ns1", "monitoring", jSlug, region, "state"},
				Source_:    hostname,
				Timestamp_: time.Now(),
				Labels_:    []core.Label{{Index: 4, Name: "job"}, {Index: 5, Name: "region"}},
				Version_:   mts[0].Version(),
			})
		}
		jobMetrics, err := client.MonitoringMetics(j.Id)
		if err != nil {
			return nil, err
		}
		for _, jm := range jobMetrics {
			for stat, m := range jm.Metrics {
				metrics = append(metrics, plugin.PluginMetricType{
					Data_:      m.Avg,
					Namespace_: []string{"raintank", "apps", "ns1", "monitoring", jSlug, jm.Region, stat},
					Source_:    hostname,
					Timestamp_: time.Now(),
					Labels_:    []core.Label{{Index: 4, Name: "job"}, {Index: 5, Name: "region"}},
					Version_:   mts[0].Version(),
				})
			}
		}
	}
	return metrics, nil
}

//GetMetricTypes returns metric types for testing
func (n *Ns1) GetMetricTypes(cfg plugin.PluginConfigType) ([]plugin.PluginMetricType, error) {
	mts := []plugin.PluginMetricType{}

	// dummy metric, so that tasks can just specify they want everything available.
	mts = append(mts, plugin.PluginMetricType{
		Namespace_: []string{"raintank", "apps", "ns1", "_all"},
		Config_:    cfg.ConfigDataNode,
	})

	mts = append(mts, plugin.PluginMetricType{
		Namespace_: []string{"raintank", "apps", "ns1", "zones", "*", "qps"},
		Labels_:    []core.Label{{Index: 4, Name: "zone"}},
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.PluginMetricType{
		Namespace_: []string{"raintank", "apps", "ns1", "monitoring", "*", "*", "state"},
		Labels_:    []core.Label{{Index: 4, Name: "job"}, {Index: 5, Name: "region"}},
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.PluginMetricType{
		Namespace_: []string{"raintank", "apps", "ns1", "monitoring", "*", "*", "rtt"},
		Labels_:    []core.Label{{Index: 4, Name: "job"}, {Index: 5, Name: "region"}},
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.PluginMetricType{
		Namespace_: []string{"raintank", "apps", "ns1", "monitoring", "*", "*", "loss"},
		Labels_:    []core.Label{{Index: 4, Name: "job"}, {Index: 5, Name: "region"}},
		Config_:    cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.PluginMetricType{
		Namespace_: []string{"raintank", "apps", "ns1", "monitoring", "*", "*", "connect"},
		Labels_:    []core.Label{{Index: 4, Name: "job"}, {Index: 5, Name: "region"}},
		Config_:    cfg.ConfigDataNode,
	})

	return mts, nil
}

//GetConfigPolicy returns a ConfigPolicyTree for testing
func (n *Ns1) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("ns1_key", true)
	p := cpolicy.NewPolicyNode()
	p.Add(rule)
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
