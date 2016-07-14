package metric_publish

import (
	"fmt"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/raintank/met"
	"github.com/raintank/worldping-api/pkg/log"
	"gopkg.in/raintank/schema.v0"
	"gopkg.in/raintank/schema.v0/msg"
)

var (
	globalProducer    *nsq.Producer
	topic             string
	metricsPublished  met.Count
	messagesPublished met.Count
	messagesSize      met.Meter
	metricsPerMessage met.Meter
	publishDuration   met.Timer
)

func Init(metrics met.Backend, t string, addr string, enabled bool) {
	if !enabled {
		return
	}
	topic = t
	cfg := nsq.NewConfig()
	cfg.UserAgent = fmt.Sprintf("raintank-apps-server")
	var err error
	globalProducer, err = nsq.NewProducer(addr, cfg)
	if err != nil {
		log.Fatal(4, "failed to initialize nsq producer. %s", err)
	}
	err = globalProducer.Ping()
	if err != nil {
		log.Fatal(4, "can't connect to nsqd: %s", err)
	}
	metricsPublished = metrics.NewCount("metricpublisher.metrics-published")
	messagesPublished = metrics.NewCount("metricpublisher.messages-published")
	messagesSize = metrics.NewMeter("metricpublisher.message_size", 0)
	metricsPerMessage = metrics.NewMeter("metricpublisher.metrics_per_message", 0)
	publishDuration = metrics.NewTimer("metricpublisher.publish_duration", 0)
}

func Publish(metrics []*schema.MetricData) error {
	if globalProducer == nil {
		log.Debug("droping %d metrics as publishing is disbaled", len(metrics))
		return nil
	}
	if len(metrics) == 0 {
		return nil
	}

	subslices := schema.Reslice(metrics, 3500)

	for _, subslice := range subslices {
		id := time.Now().UnixNano()
		data, err := msg.CreateMsg(subslice, id, msg.FormatMetricDataArrayMsgp)
		if err != nil {
			log.Fatal(4, "Fatal error creating metric message: %s", err)
		}
		metricsPublished.Inc(int64(len(subslice)))
		messagesPublished.Inc(1)
		messagesSize.Value(int64(len(data)))
		metricsPerMessage.Value(int64(len(subslice)))
		pre := time.Now()
		err = globalProducer.Publish(topic, data)
		publishDuration.Value(time.Since(pre))
		if err != nil {
			log.Fatal(4, "can't publish to nsqd: %s", err)
		}
		log.Info("published metrics %d size=%d", id, len(data))
	}

	//globalProducer.Stop()
	return nil
}
