package voxter

import (
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/grafana/metrictank/stats"
	"github.com/raintank/raintank-apps/task-agent-ng/publisher"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/schema.v1"
	log "github.com/sirupsen/logrus"
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
	voxterCollectAttemptsCount     = stats.NewCounter64("collector.voxter.collect.attempts")
	voxterCollectSuccessCount      = stats.NewCounter64("collector.voxter.collect.success")
	voxterCollectFailureCount      = stats.NewCounter64("collector.voxter.collect.failure")
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
	Publisher *publisher.Tsdb
	OrgID     int64
	Interval  int64
}

func New(task *model.TaskDTO, publisher *publisher.Tsdb) (*Voxter, error) {
	key := task.Config[task.TaskType]["voxter_key"]
	keyStr, ok := key.(string)
	if !ok {
		return nil, fmt.Errorf("voxter_key not defined in task config.")
	}
	return &Voxter{
		APIKey:    keyStr,
		Publisher: publisher,
		OrgID:     task.OrgId,
		Interval:  task.Interval,
	}, nil
}

// CollectMetrics collects metrics for testing
func (v *Voxter) CollectMetrics() {
	var err error
	if v.APIKey == "" {
		log.Error("voxter_key missing from config.")
		return
	}
	client, err := NewClient(statsURL, v.APIKey, false)
	if err != nil {
		log.Errorf("failed to create voxter api client. %s", err)
		return
	}

	resp, err := v.endpointMetrics(client)
	if err != nil {
		log.Errorf("failed to collect metrics. %s", err)
		return
	}
	if resp == nil {
		log.Error("metrics collected but no data received")
		return
	}

	if len(resp) > 0 {
		publisher.Publisher.Add(resp)
	}

	log.Debugf("collecting metrics completed. metric_count %s", len(resp))
}

func (v *Voxter) endpointMetrics(client *Client) ([]*schema.MetricData, error) {
	var metrics []*schema.MetricData
	cSlug := slug.Make("piston")
	endpoints, err := client.EndpointStats()
	if err != nil {
		return nil, err
	}
	if endpoints == nil {
		return nil, fmt.Errorf("endpoint stats collected but no data received")
	}
	for n, e := range endpoints {
		marr := strings.Split(n, ".")
		for i, v := range marr {
			marr[i] = slug.Make(v)
		}
		for i, j := 0, len(marr)-1; i < j; i, j = i+1, j-1 {
			marr[i], marr[j] = marr[j], marr[i]
		}
		mSlug := strings.Join(marr, "_")
		metrics = append(metrics, &schema.MetricData{
			OrgId:    int(v.OrgID),
			Name:     fmt.Sprintf("raintank.apps.voxter.%s.%s.registrations", cSlug, mSlug),
			Metric:   fmt.Sprintf("raintank.apps.voxter.%s.%s.registrations", cSlug, mSlug),
			Interval: int(v.Interval),
			Time:     time.Now().Unix(),
			Unit:     "ms",
			Mtype:    "gauge",
			Value:    e.Registrations,
			Tags:     nil,
		}, &schema.MetricData{
			OrgId:    int(v.OrgID),
			Name:     fmt.Sprintf("raintank.apps.voxter.%s.%s.channels.inbound", cSlug, mSlug),
			Metric:   fmt.Sprintf("raintank.apps.voxter.%s.%s.channels.inbound", cSlug, mSlug),
			Interval: int(v.Interval),
			Time:     time.Now().Unix(),
			Unit:     "ms",
			Mtype:    "gauge",
			Value:    e.Channels.Inbound,
			Tags:     nil,
		}, &schema.MetricData{
			OrgId:    int(v.OrgID),
			Name:     fmt.Sprintf("raintank.apps.voxter.%s.%s.channels.outbound", cSlug, mSlug),
			Metric:   fmt.Sprintf("raintank.apps.voxter.%s.%s.channels.outbound", cSlug, mSlug),
			Interval: int(v.Interval),
			Time:     time.Now().Unix(),
			Unit:     "ms",
			Mtype:    "gauge",
			Value:    e.Channels.Outbound,
			Tags:     nil,
		})
	}
	for _, m := range metrics {
		m.SetId()
	}

	return metrics, nil
}
