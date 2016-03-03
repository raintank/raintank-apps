package hostedtsdb

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"

	"github.com/raintank/raintank-metric/schema"
)

const (
	name                 = "rt-hostedtsdb"
	version              = 1
	pluginType           = plugin.PublisherPluginType
	maxMetricsPerPayload = 3000
)

type HostedtsdbPublisher struct {
}

func NewHostedtsdbPublisher() *HostedtsdbPublisher {
	return &HostedtsdbPublisher{}
}

type Queue struct {
	sync.Mutex
	RemoteUrl *url.URL
	Token     string
	Metrics   []*schema.MetricData
}

func (q *Queue) Add(metrics []*schema.MetricData) {
	q.Lock()
	q.Metrics = append(q.Metrics, metrics...)
	flush := false
	if len(q.Metrics) > maxMetricsPerPayload {
		flush = true
	}
	q.Unlock()
	if flush {
		q.Flush()
	}
}

func (q *Queue) Flush() {
	q.Lock()
	metrics := make([]*schema.MetricData, len(q.Metrics))
	copy(metrics, q.Metrics)
	q.Metrics = q.Metrics[:0]
	q.Unlock()
	// Write the metrics to our HTTP server.
	log.Printf("writing %d metrics to API", len(metrics))
}

type WriteQueue struct {
	sync.Mutex
	Queues    map[string]*Queue
	QueueFull chan string
}

func (w *WriteQueue) Run() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			w.FlushAll()
		case key := <-w.QueueFull:
			w.Lock()
			if q, ok := w.Queues[key]; ok {
				go q.Flush()
			}
			w.Unlock()
		}
	}
}

func (w *WriteQueue) FlushAll() {
	w.Lock()
	for _, q := range w.Queues {
		go q.Flush()
	}
	w.Unlock()
}

func (w *WriteQueue) Add(metrics []*schema.MetricData, url *url.URL, token string) {
	qKey := fmt.Sprintf("%s:%s", token, url.String())
	var q *Queue
	var ok bool
	w.Lock()
	if q, ok = w.Queues[qKey]; !ok {
		q = &Queue{
			RemoteUrl: url,
			Token:     token,
			Metrics:   make([]*schema.MetricData, 0),
		}
		w.Queues[qKey] = q
	}
	w.Unlock()
	q.Add(metrics)
}

func NewWriteQueue() *WriteQueue {
	return &WriteQueue{
		Queues:    make(map[string]*Queue),
		QueueFull: make(chan string, 10),
	}
}

type EventQueue struct {
}

func (e *EventQueue) Add(m *plugin.PluginMetricType) error {
	return nil
}

func NewEventQueue() *EventQueue {
	return &EventQueue{}
}

var writeQueue *WriteQueue

var eventQueue *EventQueue

func init() {
	writeQueue = NewWriteQueue()
	go writeQueue.Run()
	eventQueue = NewEventQueue()
	//go eventQueue.Run()
}

func (f *HostedtsdbPublisher) Publish(contentType string, content []byte, config map[string]ctypes.ConfigValue) error {
	log.Println("Publishing started")
	var metrics []plugin.PluginMetricType

	switch contentType {
	case plugin.SnapGOBContentType:
		dec := gob.NewDecoder(bytes.NewBuffer(content))
		if err := dec.Decode(&metrics); err != nil {
			log.Printf("Error decoding: error=%v content=%v", err, content)
			return err
		}
	default:
		log.Printf("Error unknown content type '%v'", contentType)
		return errors.New(fmt.Sprintf("Unknown content type '%s'", contentType))
	}

	log.Printf("publishing %v metrics to %v", len(metrics), config)
	remoteUrl, err := url.Parse(config["url"].(ctypes.ConfigValueStr).Value)
	if err != nil {
		return err
	}
	token := config["token"].(ctypes.ConfigValueStr).Value
	interval := config["interval"].(ctypes.ConfigValueInt).Value

	metricsArray := make([]*schema.MetricData, len(metrics))
	for i, m := range metrics {
		tags := make([]string, 0)
		targetType := "gauge"
		unit := ""
		for k, v := range m.Tags() {
			switch k {
			case "targetType":
				targetType = v
			case "unit":
				unit = v
			default:
				tags = append(tags, fmt.Sprintf("%s:%s", k, v))
			}
		}
		var value float64
		rawData := m.Data()
		switch rawData.(type) {
		case int:
			value = float64(rawData.(int))
		case int8:
			value = float64(rawData.(int8))
		case int16:
			value = float64(rawData.(int16))
		case int32:
			value = float64(rawData.(int32))
		case int64:
			value = float64(rawData.(int64))
		case uint8:
			value = float64(rawData.(uint8))
		case uint16:
			value = float64(rawData.(uint16))
		case uint32:
			value = float64(rawData.(uint32))
		case uint64:
			value = float64(rawData.(uint64))
		case float32:
			value = float64(rawData.(float32))
		case float64:
			value = rawData.(float64)
		case string:
			//payload is an event.
			return eventQueue.Add(&m)
		default:
			return errors.New("unknown data type")
		}

		metricsArray[i] = &schema.MetricData{
			OrgId:      1,
			Name:       strings.Join(m.Namespace(), "."),
			Interval:   interval,
			Value:      value,
			Time:       m.Timestamp().Unix(),
			TargetType: targetType,
			Unit:       unit,
			Tags:       tags,
		}
	}
	writeQueue.Add(metricsArray, remoteUrl, token)

	return nil
}

func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(
		name,
		version,
		pluginType,
		[]string{plugin.SnapGOBContentType},
		[]string{plugin.SnapGOBContentType},
		plugin.Exclusive(true),
	)
}

func (f *HostedtsdbPublisher) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("url", true)
	rule2, _ := cpolicy.NewStringRule("token", true)
	rule3, _ := cpolicy.NewIntegerRule("interval", true)
	rule4, _ := cpolicy.NewIntegerRule("orgId", true)

	p := cpolicy.NewPolicyNode()
	p.Add(rule)
	p.Add(rule2)
	p.Add(rule3)
	p.Add(rule4)
	c.Add([]string{""}, p)
	return c, nil
}

func handleErr(e error) {
	if e != nil {
		panic(e)
	}
}
