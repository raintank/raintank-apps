// Package ns1 provides a custom plugin to query the NS1 api and return metrics that can be sent to metrictank
package ns1

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

var (
	ns1CollectStatsAttempts  = stats.NewCounter64("collector.ns1.collect.attempts.count")
	ns1CollectStatsSucceeded = stats.NewCounter64("collector.ns1.collect.success.count")
	ns1CollectStatsFailures  = stats.NewCounter64("collector.ns1.collect.failures.count")
)
var (
	statusMap = map[string]int{"up": 0, "down": 1}
)

func init() {
	slug.CustomSub = map[string]string{".": "_"}
}

// Ns1 Plugin Name
type Ns1 struct {
	APIKey string
	Metric *taskrunner.RTAMetric
}

// CollectMetrics collects metrics for testing
func (n *Ns1) CollectMetrics() {
	var err error
	if n.APIKey == "" {
		log.Error(4, "ns1_key missing from config.")
		return
	}
	client, err := NewClient("https://api.nsone.net/", n.APIKey, false)
	if err != nil {
		log.Error(4, "failed to create NS1 api client: error ", err)
		return
	}
	result, probeErr := n.zoneMetrics(client, n.Metric)
	if probeErr != nil {
		log.Error(4, "failed to collect metrics.", probeErr)
		return
	}
	log.Info("QPS is ", result.Value)
	spew.Dump(result)
	zoneSlug := slug.Make(n.Metric.Zone)

	var metrics []*schema.MetricData
	qpsMetric := schema.MetricData{
		Id:       "1",
		OrgId:    1,
		Name:     fmt.Sprintf("raintank.apps.ns1.zones.%s.qps", zoneSlug),
		Metric:   fmt.Sprintf("raintank.apps.ns1.zones.%s.qps", zoneSlug),
		Interval: 60,
		Time:     time.Now().Unix(),
		Unit:     "ms",
		Mtype:    "gauge",
		Value:    result.Value,
		Tags:     nil,
	}
	metrics = append(metrics, &qpsMetric)
	log.Debug("got %d metrics", len(metrics))
	spew.Dump(metrics)

	publisher.Publisher.Add(metrics)
	log.Debug("collecting metrics completed")
}

func (n *Ns1) zoneMetrics(client *Client, metric *taskrunner.RTAMetric) (*taskrunner.RTAMetric, error) {
	//zSlug := slug.Make(mt.Zone)
	ns1CollectStatsAttempts.Inc()
	qps, err := client.QPS(metric.Zone)
	if err != nil {
		log.Error(4, "failed to get zone QPS for zone - %d error %s", metric.Zone, err)
		ns1CollectStatsFailures.Inc()
	} else {
		metric.Value = qps.QPS
		metric.Timestamp = time.Now().Unix()
		ns1CollectStatsSucceeded.Inc()
	}
	return metric, nil
}
