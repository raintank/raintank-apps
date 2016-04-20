package ping

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
)

const (
	// Name of plugin
	Name = "worldping-ping"
	// Version of plugin
	Version = 1
	// Type of plugin
	Type = plugin.CollectorPluginType
)

type StateCache struct {
	mu     sync.Mutex
	Checks map[string]int
}

func (s *StateCache) Get(key string) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.Checks[key]
	return val, ok
}
func (s *StateCache) Set(key string, value int) {
	s.mu.Lock()
	s.Checks[key] = value
	s.mu.Unlock()
	return
}

var (
	// make sure that we actually satisify requierd interface
	_ plugin.CollectorPlugin = (*Ping)(nil)

	stateCache *StateCache

	metricNames = []string{
		"avg",
		"min",
		"max",
		"median",
		"mdev",
		"loss",
	}
)

func init() {
	stateCache = &StateCache{Checks: make(map[string]int)}
}

type Ping struct {
}

// CollectMetrics collects metrics for testing
func (p *Ping) CollectMetrics(mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
	var err error

	conf := mts[0].Config().Table()
	checkId, ok := conf["checkId"]
	if !ok || checkId.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("checkId missing from config, %v", conf)
	}
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

	metrics, err := ping(checkId.(ctypes.ConfigValueStr).Value, agentName.(ctypes.ConfigValueStr).Value, endpoint.(ctypes.ConfigValueStr).Value, hostname.(ctypes.ConfigValueStr).Value, mts)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func ping(checkId, agentName, endpoint, host string, mts []plugin.PluginMetricType) ([]plugin.PluginMetricType, error) {
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
	for _, m := range mts {
		stat := m.Namespace()[4]
		if value, ok := stats[stat]; ok {
			mt := plugin.PluginMetricType{
				Data_:      value,
				Namespace_: []string{"worlding", agentName, endpoint, "ping", stat},
				Source_:    hostname,
				Timestamp_: time.Now(),
				Labels_:    m.Labels(),
				Version_:   m.Version(),
			}
			metrics = append(metrics, mt)
		}
	}

	//check if state has changed.
	state := 0
	if check.Result.Error != nil {
		state = 1
	}

	lastState, ok := stateCache.Get(checkId)
	if !ok {
		lastState = -1
	}
	stateCache.Set(checkId, state)

	if state != lastState {
		var stat, message string

		if state == 0 {
			message = "Monitor now OK"
			stat = "OK"
		} else {
			message = *check.Result.Error
			stat = "ERROR"
		}
		mt := plugin.PluginMetricType{
			Data_:      message,
			Namespace_: []string{"worlding", "event", "monitor_state", stat},
			Tags_:      map[string]string{"endpoint": endpoint, "probe": agentName, "monitor_type": "ping"},
			Source_:    hostname,
			Timestamp_: time.Now(),
			Version_:   mts[0].Version(),
		}
		metrics = append(metrics, mt)
	}

	return metrics, nil
}

//GetMetricTypes returns metric types for testing
func (p *Ping) GetMetricTypes(cfg plugin.PluginConfigType) ([]plugin.PluginMetricType, error) {
	mts := []plugin.PluginMetricType{}
	for _, metricName := range metricNames {
		mts = append(mts, plugin.PluginMetricType{
			Namespace_: []string{"worldping", "*", "*", "ping", metricName},
			Labels_:    []core.Label{{Index: 1, Name: "endpoint"}, {Index: 2, Name: "probe"}},
		})
	}
	return mts, nil
}

//GetConfigPolicy returns a ConfigPolicyTree for testing
func (p *Ping) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule0, _ := cpolicy.NewStringRule("checkId", true)
	rule1, _ := cpolicy.NewStringRule("endpoint", true)
	rule2, _ := cpolicy.NewStringRule("raintank_agent_name", true)
	rule3, _ := cpolicy.NewStringRule("hostname", true)
	cp := cpolicy.NewPolicyNode()
	cp.Add(rule0)
	cp.Add(rule1)
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
