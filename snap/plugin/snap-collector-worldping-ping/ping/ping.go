package ping

import (
	"fmt"
	"os"
	"time"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
	"github.com/raintank/go-pinger"
)

const (
	// Name of plugin
	Name = "worldping-ping"
	// Version of plugin
	Version = 1
	// Type of plugin
	Type = plugin.CollectorPluginType
)

// make sure that we actually satisify requierd interface
var _ plugin.CollectorPlugin = (*Ping)(nil)

func init() {
	GlobalPinger = pinger.NewPinger()
}

type Ping struct {
}

// CollectMetrics collects metrics for testing
func (p *Ping) CollectMetrics(mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	var err error

	if len(mts) != 1 {
		return nil, fmt.Errorf("only 1 pluginMetricType supported.")
	}
	conf := mts[0].Config().Table()
	hostname, ok := conf["hostname"]
	if !ok || hostname.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("hostname missing from config, %v", conf)
	}
	endpoint, ok := conf["endpoint"]
	if !ok || endpoint.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("endpoint missing from config, %v", conf)
	}
	agentName, ok := conf["raintank_agent_name"]
	if !ok || agentName.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("raintank_agent_name missing from config, %v", conf)
	}

	metrics, err := ping(agentName.(ctypes.ConfigValueStr).Value, hostname.(ctypes.ConfigValueStr).Value, hostname.(ctypes.ConfigValueStr).Value, mts)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func ping(agentName, endpoint, host string, mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	hostname, _ := os.Hostname()
	check := &RaintankProbePing{
		Hostname: host,
		Timeout:  10,
	}
	err := check.Run()
	if err != nil {
		return nil, err
	}
	stats := make(map[string]float64)
	if check.Result.Avg != nil {
		stats["avg"] = *check.Result.Avg
	}
	if check.Result.Min != nil {
		stats["min"] = *check.Result.Min
	}
	if check.Result.Max != nil {
		stats["max"] = *check.Result.Max
	}
	if check.Result.Median != nil {
		stats["median"] = *check.Result.Median
	}
	if check.Result.Mdev != nil {
		stats["mdev"] = *check.Result.Mdev
	}
	if check.Result.Loss != nil {
		stats["loss"] = *check.Result.Loss
	}

	metrics := make([]plugin.PluginMetricType, 0, len(stats))
	for stat, value := range stats {
		mt := plugin.PluginMetricType{
			Data_:      value,
			Namespace_: []string{"worlding", agentName, endpoint, "ping", stat},
			Source_:    hostname,
			Timestamp_: time.Now(),
			Labels_:    mts[0].Labels(),
			Version_:   mts[0].Version(),
		}
		metrics = append(metrics, mt)
	}

	return metrics, nil
}

//GetMetricTypes returns metric types for testing
func (p *Ping) GetMetricTypes(cfg plugin.PluginConfigType) ([]plugin.PluginMetricType, error) {
	mts := []plugin.PluginMetricType{}
	mts = append(mts, plugin.PluginMetricType{
		Namespace_: []string{"worldping", "*", "*", "ping", "*"},
		Labels_:    []core.Label{{Index: 1, Name: "endpoint"}, {Index: 2, Name: "probe"}, {Index: 3, Name: "stat"}},
	})
	return mts, nil
}

//GetConfigPolicy returns a ConfigPolicyTree for testing
func (p *Ping) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("endpoint", true)
	rule2, _ := cpolicy.NewStringRule("raintank_agent_name", true)
	rule3, _ := cpolicy.NewStringRule("hostname", true)
	cp := cpolicy.NewPolicyNode()
	cp.Add(rule)
	cp.Add(rule2)
	cp.Add(rule3)
	c.Add([]string{"worldping"}, cp)
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
		plugin.ConcurrencyCount(5000),
	)
}
