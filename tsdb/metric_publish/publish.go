package metric_publish

import (
	"fmt"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/op/go-logging"
	"github.com/raintank/met"
	msg "github.com/raintank/raintank-metric/msg"
	"github.com/raintank/raintank-metric/schema"
)

var log = logging.MustGetLogger("default")

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
		log.Fatalf("failed to initialize nsq producer. %s", err)
	}
	err = globalProducer.Ping()
	if err != nil {
		log.Fatalf("can't connect to nsqd: %s", err)
	}
	metricsPublished = metrics.NewCount("metricpublisher.metrics-published")
	messagesPublished = metrics.NewCount("metricpublisher.messages-published")
	messagesSize = metrics.NewMeter("metricpublisher.message_size", 0)
	metricsPerMessage = metrics.NewMeter("metricpublisher.metrics_per_message", 0)
	publishDuration = metrics.NewTimer("metricpublisher.publish_duration", 0)
}

func Publish(metrics []*schema.MetricData) error {
	if globalProducer == nil {
		log.Debugf("droping %d metrics as publishing is disbaled", len(metrics))
		return nil
	}
	if len(metrics) == 0 {
		return nil
	}

	id := time.Now().UnixNano()
	data, err := msg.CreateMsg(metrics, id, msg.FormatMetricDataArrayMsgp)
	if err != nil {
		log.Fatal(0, "Fatal error creating metric message: %s", err)
	}
	metricsPublished.Inc(int64(len(metrics)))
	messagesPublished.Inc(1)
	messagesSize.Value(int64(len(data)))
	metricsPerMessage.Value(int64(len(metrics)))
	pre := time.Now()
	err = globalProducer.Publish(topic, data)
	publishDuration.Value(time.Since(pre))
	if err != nil {
		log.Fatal(0, "can't publish to nsqd: %s", err)
	}
	log.Info("published metrics %d size=%d", id, len(data))

	//globalProducer.Stop()
	return nil
}
