package event_publish

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
	globalProducer  *nsq.Producer
	topic           string
	eventsPublished met.Count
	messagesSize    met.Meter
	publishDuration met.Timer
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
		log.Fatal(4, "failed to initialize nsq producer for events. %s", err)
	}
	err = globalProducer.Ping()
	if err != nil {
		log.Fatal(4, "can't connect to nsqd: %s", err)
	}
	eventsPublished = metrics.NewCount("eventpublisher.events-published")
	messagesSize = metrics.NewMeter("eventpublisher.message_size", 0)
	publishDuration = metrics.NewTimer("eventpublisher.publish_duration", 0)
}

func Publish(event *schema.ProbeEvent) error {
	if globalProducer == nil {
		log.Debug("droping event as publishing is disbaled")
		return nil
	}

	id := time.Now().UnixNano()
	data, err := msg.CreateProbeEventMsg(event, id, msg.FormatProbeEventMsgp)
	if err != nil {
		log.Fatal(4, "Fatal error creating event message: %s", err)
	}
	eventsPublished.Inc(1)
	messagesSize.Value(int64(len(data)))
	pre := time.Now()
	err = globalProducer.Publish(topic, data)
	publishDuration.Value(time.Since(pre))
	if err != nil {
		log.Fatal(4, "can't publish to nsqd: %s", err)
	}
	log.Debug("published event %d", id)

	return nil
}
