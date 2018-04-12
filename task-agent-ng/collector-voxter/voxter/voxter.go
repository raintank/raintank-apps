package voxter

import (
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gosimple/slug"
	"github.com/grafana/metrictank/stats"
	"github.com/raintank/raintank-apps/task-agent-ng/publisher"
	"github.com/raintank/raintank-apps/task-agent-ng/taskrunner"
	"github.com/raintank/schema.v1"
	"github.com/raintank/worldping-api/pkg/log"
)

const (
	// Name of plugin
	Name = "voxter"
	// Version of plugin
	Version = 2
	// stat url
	statsURL = "https://vortex2.voxter.com/api/"
)

var (
	voxterCollectAttemptsCount     = stats.NewCounter64("collector.voxter.collect.attempts.count")
	voxterCollectSuccessCount      = stats.NewCounter64("collector.voxter.collect.success.count")
	voxterCollectFailureCount      = stats.NewCounter64("collector.voxter.collect.failure.count")
	voxterCollectDurationNS        = stats.NewGauge64("collector.voxter.collect.duration_ns")
	voxterCollectSuccessDurationNS = stats.NewGauge64("collector.voxter.collect.success.duration_ns")
	voxterCollectFailureDurationNS = stats.NewGauge64("collector.voxter.collect.failure.duration_ns")
)

var (
	statusMap = map[string]int{"up": 0, "down": 1}
)

func init() {
	slug.CustomSub = map[string]string{".": "_"}
}

type Voxter struct {
	APIKey    string
	Metric    *taskrunner.RTAMetric
	Publisher *publisher.Tsdb
	OrgID     int64
	Interval  int64
}

// CollectMetrics collects metrics for testing
func (v *Voxter) CollectMetrics() error {
	var err error
	if v.APIKey == "" {
		log.Error(4, "voxter_key missing from config.")
		return fmt.Errorf("voxter_key missing from config")
	}
	client, err := NewClient(statsURL, v.APIKey, false)
	if err != nil {
		log.Error(4, "failed to create voxter api client.", "error", err)
		return err
	}

	resp, err := v.endpointMetrics(client)
	if err != nil {
		log.Error(4, "failed to collect metrics.", "error", err)
		return err
	}
	if resp == nil {
		return fmt.Errorf("metrics collected but no data received")
	}
	//metrics = resp
	aMetric := schema.MetricData{
		Id:       "1",
		OrgId:    int(v.OrgID),
		Name:     fmt.Sprintf("raintank.apps.voxter.%s.ametric", ""),
		Metric:   fmt.Sprintf("raintank.apps.voxter.%s.ametric", ""),
		Interval: int(v.Interval),
		Time:     time.Now().Unix(),
		Unit:     "ms",
		Mtype:    "gauge",
		Value:    0,
		Tags:     nil,
	}
	var metrics []*schema.MetricData

	metrics = append(metrics, &aMetric)
	log.Debug("got %d metrics", len(metrics))
	spew.Dump(metrics)

	publisher.Publisher.Add(metrics)

	log.Debug("collecting metrics completed", "metric_count", len(metrics))
	return nil
}

func (v *Voxter) endpointMetrics(client *Client) ([]*schema.MetricData, error) {
	var metrics []*schema.MetricData
	endpoints, err := client.EndpointStats()
	if err != nil {
		return nil, err
	}
	if endpoints == nil {
		return nil, fmt.Errorf("endpoint stats collected but no data received")
	}
	// TODO

	return metrics, nil
}
