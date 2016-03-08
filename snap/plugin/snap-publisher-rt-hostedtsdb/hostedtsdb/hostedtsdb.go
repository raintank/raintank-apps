package hostedtsdb

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"

	"github.com/raintank/raintank-metric/msg"
	"github.com/raintank/raintank-metric/schema"
)

const (
	name                 = "rt-hostedtsdb"
	version              = 1
	pluginType           = plugin.PublisherPluginType
	maxMetricsPerPayload = 3000
)

var (
	RemoteUrl *url.URL
	Token     string
)

type HostedtsdbPublisher struct {
}

func NewHostedtsdbPublisher() *HostedtsdbPublisher {
	return &HostedtsdbPublisher{}
}

type WriteQueue struct {
	sync.Mutex
	Metrics   []*schema.MetricData
	QueueFull chan struct{}
}

func (q *WriteQueue) Add(metrics []*schema.MetricData) {
	q.Lock()
	q.Metrics = append(q.Metrics, metrics...)
	if len(q.Metrics) > maxMetricsPerPayload {
		q.QueueFull <- struct{}{}
	}
	q.Unlock()
}

func (q *WriteQueue) Flush() {
	q.Lock()
	if len(q.Metrics) == 0 {
		q.Unlock()
		return
	}
	metrics := make([]*schema.MetricData, len(q.Metrics))
	copy(metrics, q.Metrics)
	q.Metrics = q.Metrics[:0]
	q.Unlock()
	// Write the metrics to our HTTP server.
	log.Printf("writing %d metrics to API", len(metrics))
	id := time.Now().UnixNano()
	body, err := msg.CreateMsg(metrics, id, msg.FormatMetricDataArrayMsgp)
	if err != nil {
		log.Printf("Error: unable to convert metrics to MetricDataArrayMsgp. %s", err)
		return
	}
	sent := false
	for !sent {
		if err = PostData(RemoteUrl, Token, body); err != nil {
			log.Printf("Error: %s", err)
			time.Sleep(time.Second)
		} else {
			sent = true
		}
	}
}

func (q *WriteQueue) Run() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			q.Flush()
		case <-q.QueueFull:
			q.Flush()
		}
	}
}

func NewWriteQueue() *WriteQueue {
	return &WriteQueue{
		Metrics:   make([]*schema.MetricData, 0),
		QueueFull: make(chan struct{}),
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

	// set the RemoteURL and Token when the first metrics is recieved.
	var err error
	if RemoteUrl == nil {
		RemoteUrl, err = url.Parse(config["raintank_tsdb_url"].(ctypes.ConfigValueStr).Value)
		if err != nil {
			return err
		}
	}
	if Token == "" {
		Token = config["raintank_api_key"].(ctypes.ConfigValueStr).Value
	}
	//-----------------

	interval := config["interval"].(ctypes.ConfigValueInt).Value
	orgId := config["orgId"].(ctypes.ConfigValueInt).Value

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
			OrgId:      orgId,
			Name:       strings.Join(m.Namespace(), "."),
			Interval:   interval,
			Value:      value,
			Time:       m.Timestamp().Unix(),
			TargetType: targetType,
			Unit:       unit,
			Tags:       tags,
		}
	}
	writeQueue.Add(metricsArray)

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
	rule, _ := cpolicy.NewStringRule("raintank_tsdb_url", true)
	rule2, _ := cpolicy.NewStringRule("raintank_api_key", true)
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

func PostData(remoteUrl *url.URL, token string, body []byte) error {
	req, err := http.NewRequest("POST", remoteUrl.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "rt-metric-binary")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	respBody, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Posting data failed. %d - %s", resp.StatusCode, string(respBody))
	}
	return nil
}
