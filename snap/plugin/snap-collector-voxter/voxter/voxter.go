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
	// stat url
	statsURL = "http://ourstatshost.voxter.com/api/"
)

var (
	statusMap = map[string]int{"up": 0, "down": 1}
)

func init() {
	slug.CustomSub = map[string]string{".": "_"}
}

// make sure that we actually satisify required interface
var _ plugin.CollectorPlugin = (*Voxter)(nil)

type Voxter struct {
}

// CollectMetrics collects metrics for testing
func (v *Voxter) CollectMetrics(mts []plugin.MetricType) ([]plugin.MetricType, error) {
	var err error
	metrics := make([]plugin.MetricType, 0)
	conf := mts[0].Config().Table()
	apiKey, ok := conf["voxter_key"]
	if !ok || apiKey.(ctypes.ConfigValueStr).Value == "" {
		LogError("voxter_key missing from config.")
		return nil, fmt.Errorf("voxter_key missing from config, %v", conf)
	}
	client, err := NewClient(statsURL, apiKey.(ctypes.ConfigValueStr).Value, false)
	if err != nil {
		LogError("failed to create voxter api client.", "error", err)
		return nil, err
	}
	LogDebug("request to collect metrics", "metric_count", len(mts))

	resp, err := v.EndpointMetrics(client, mts)
	if err != nil {
		LogError("failed to collect metrics.", "error", err)
		return nil, err
	}
	metrics = resp

	LogDebug("collecting metrics completed", "metric_count", len(metrics))
	return metrics, nil
}

//GetMetricTypes returns metric types for testing
func (v *Voxter) GetMetricTypes(cfg plugin.ConfigType) ([]plugin.MetricType, error) {
	mts := []plugin.MetricType{}

	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "voxter", "endpoints", "*", "*", "registrations"),
		Config_: cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "voxter", "endpoints", "*", "*", "channels", "inbound"),
		Config_: cfg.ConfigDataNode,
	})
	mts = append(mts, plugin.MetricType{
		Namespace_: core.NewNamespace("raintank", "apps", "voxter", "endpoints", "*", "*", "channels", "outbound"),
		Config_: cfg.ConfigDataNode,
	})

	return mts, nil
}

func (v *Voxter) EndpointMetrics(client *Client, mts []plugin.MetricType) ([]plugin.MetricType, error) {
	var metrics []plugin.MetricType
	conf := mts[0].Config().Table()
	cust, ok := conf["customer"]
	if !ok || cust.(ctypes.ConfigValueStr).Value == "" {
		LogError("customer missing from config")
		return metrics, nil
	}
	cName := cust.(ctypes.ConfigValueStr).Value
	cSlug := slug.Make(cName)
	endpoints, err := client.EndpointStats(cName)
	if err != nil {
		return nil, err
	}
	metrics = make([]plugin.MetricType, len(endpoints) * 3)
	for _, e := range endpoints {
		mSlug := slug.Make(e.Name)
		metrics = append(metrics, plugin.MetricType{
			Data_: e.Registrations,
			Namespace_: core.NewNamespace("raintank", "apps", "voxter", "endpoints", cSlug, mSlug, "registrations"),
			Timestamp_: time.Now(),
			Version_: mts[0].Version(),
		})
		metrics = append(metrics, plugin.MetricType{
			Data_: e.Channels.Inbound,
			Namespace_: core.NewNamespace("raintank", "apps", "voxter", "endpoints", cSlug, mSlug, "channels", "inbound"),
			Timestamp_: time.Now(),
			Version_: mts[0].Version(),
		})
		metrics = append(metrics, plugin.MetricType{
			Data_: e.Channels.Outbound,
			Namespace_: core.NewNamespace("raintank", "apps", "voxter", "endpoints", cSlug, mSlug, "channels", "outbound"),
			Timestamp_: time.Now(),
			Version_: mts[0].Version(),
		})
	}

	return metrics, nil
}

//GetConfigPolicy returns a ConfigPolicyTree for testing
func (v *Voxter) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("voxter_key", true)
	p := cpolicy.NewPolicyNode()
	p.Add(rule)

	c.Add([]string{"raintank", "apps", "voxter"}, p)
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
