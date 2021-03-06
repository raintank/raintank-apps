// Package ns1 provides a custom plugin to query the NS1 api and return metrics that can be sent to metrictank
package ns1

import (
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"github.com/grafana/metrictank/stats"
	"github.com/raintank/raintank-apps/task-agent-ng/publisher"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/schema.v1"
	log "github.com/sirupsen/logrus"
)

var (
	ns1CollectAttemptsCount     = stats.NewCounter64("collector.ns1.collect.attempts")
	ns1CollectSuccessCount      = stats.NewCounter64("collector.ns1.collect.success")
	ns1CollectFailureCount      = stats.NewCounter64("collector.ns1.collect.failure")
	ns1CollectDurationNS        = stats.NewGauge64("collector.ns1.collect.duration_ns")
	ns1CollectSuccessDurationNS = stats.NewGauge64("collector.ns1.collect.success.duration_ns")
	ns1CollectFailureDurationNS = stats.NewGauge64("collector.ns1.collect.failure.duration_ns")
)
var (
	statusMap = map[string]int{"up": 0, "down": 1}
)

func init() {
	slug.CustomSub = map[string]string{".": "_"}
}

// Ns1 Plugin Name
type Ns1 struct {
	APIKey    string
	Zone      string
	Publisher *publisher.Tsdb
	OrgID     int64
	Interval  int64
}

func New(task *model.TaskDTO, publisher *publisher.Tsdb) (*Ns1, error) {
	key := task.Config[task.TaskType]["ns1_key"]
	keyStr, ok := key.(string)
	if !ok {
		return nil, fmt.Errorf("ns1_key not defined in task config.")
	}
	zone := task.Config[task.TaskType]["zone"]
	zoneStr, ok := zone.(string)
	if !ok {
		return nil, fmt.Errorf("zone not defined in task config.")
	}
	return &Ns1{
		APIKey:    keyStr,
		Zone:      zoneStr,
		Publisher: publisher,
		OrgID:     task.OrgId,
		Interval:  task.Interval,
	}, nil
}

// CollectMetrics collects metrics for testing
func (n *Ns1) CollectMetrics() {
	var err error
	if n.APIKey == "" {
		log.Error("ns1_key missing from config.")
		return
	}
	client, err := NewClient("https://api.nsone.net/", n.APIKey, false)
	if err != nil {
		log.Errorf("failed to create NS1 api client: %s", err)
		return
	}
	result, err := n.zoneMetrics(client, n.Zone)
	if err != nil {
		log.Errorf("failed to collect metrics. %s", err)
		return
	}
	log.Infof("QPS for %s is %f", n.Zone, result)
	zoneSlug := slug.Make(n.Zone)

	metrics := make([]*schema.MetricData, 1)
	metrics[0] = &schema.MetricData{
		OrgId:    int(n.OrgID),
		Name:     fmt.Sprintf("raintank.apps.ns1.zones.%s.qps", zoneSlug),
		Metric:   fmt.Sprintf("raintank.apps.ns1.zones.%s.qps", zoneSlug),
		Interval: int(n.Interval),
		Time:     time.Now().Unix(),
		Unit:     "ms",
		Mtype:    "gauge",
		Value:    result,
		Tags:     nil,
	}
	metrics[0].SetId()

	// publish to tsdbgw
	n.Publisher.Add(metrics)
	log.Debug("collecting metrics completed")
}

func (n *Ns1) zoneMetrics(client *Client, zone string) (float64, error) {
	ns1CollectAttemptsCount.Inc()
	startTime := time.Now().UTC()
	qps, err := client.QPS(zone)
	result := float64(0)
	if err != nil {
		log.Errorf("failed to get zone QPS for zone - %s error %s", zone, err)
		ns1CollectFailureCount.Inc()
		endTime := time.Since(startTime)
		ns1CollectFailureDurationNS.SetUint64(uint64(endTime.Nanoseconds()))
		ns1CollectDurationNS.SetUint64(uint64(endTime.Nanoseconds()))
	} else {
		result = qps.QPS
		ns1CollectSuccessCount.Inc()
		endTime := time.Since(startTime)
		ns1CollectSuccessDurationNS.SetUint64(uint64(endTime.Nanoseconds()))
		ns1CollectDurationNS.SetUint64(uint64(endTime.Nanoseconds()))
	}
	return result, nil
}
