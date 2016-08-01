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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core/ctypes"

	"gopkg.in/raintank/schema.v1"
	"gopkg.in/raintank/schema.v1/msg"
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
		if err = PostData("metrics", Token, body); err != nil {
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

var writeQueue *WriteQueue

func init() {
	writeQueue = NewWriteQueue()
	go writeQueue.Run()
}

func (f *HostedtsdbPublisher) Publish(contentType string, content []byte, config map[string]ctypes.ConfigValue) error {
	log.Println("Publishing started")
	var metrics []plugin.MetricType

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

	log.Printf("publishing %d metrics to %v", len(metrics), config)

	// set the RemoteURL and Token when the first metrics is recieved.
	var err error
	if RemoteUrl == nil {
		remote := config["raintank_tsdb_url"].(ctypes.ConfigValueStr).Value
		if !strings.HasSuffix(remote, "/") {
			remote += "/"
		}
		RemoteUrl, err = url.Parse(remote)
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
		var value float64
		rawData := m.Data()
		switch rawData.(type) {
		case string:
			//payload is an event.
			go sendEvent(int64(orgId), &m)
			continue
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
		default:
			return errors.New("unknown data type")
		}

		tags := make([]string, 0)
		mtype := "gauge"
		unit := ""
		for k, v := range m.Tags() {
			switch k {
			case "mtype":
				mtype = v
			case "unit":
				unit = v
			default:
				tags = append(tags, fmt.Sprintf("%s:%s", k, v))
			}
		}

		metricsArray[i] = &schema.MetricData{
			OrgId:    orgId,
			Name:     m.Namespace().Key(),
			Interval: interval,
			Value:    value,
			Time:     m.Timestamp().Unix(),
			Mtype:    mtype,
			Unit:     unit,
			Tags:     tags,
		}
		metricsArray[i].SetId()
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
		plugin.ConcurrencyCount(1000),
	)
}

func (f *HostedtsdbPublisher) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("raintank_tsdb_url", true)
	rule2, _ := cpolicy.NewStringRule("raintank_api_key", true)
	rule3, _ := cpolicy.NewIntegerRule("interval", true)
	rule4, _ := cpolicy.NewIntegerRule("orgId", false, 0)

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

func PostData(path, token string, body []byte) error {
	u := RemoteUrl.String() + path
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "rt-metric-binary")
	req.Header.Set("Authorization", "Bearer "+token)

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

func sendEvent(orgId int64, m *plugin.MetricType) {
	ns := m.Namespace().Strings()
	if len(ns) != 4 {
		log.Printf("Error: invalid event metric. Expected namesapce to be 4 fields.")
		return
	}
	if ns[0] != "worldping" || ns[1] != "event" {
		log.Printf("Error: invalid event metrics.  Metrics hould begin with 'worldping.event'")
		return
	}
	hostname, _ := os.Hostname()
	id := time.Now().UnixNano()
	event := &schema.ProbeEvent{
		OrgId:     orgId,
		EventType: ns[2],
		Severity:  ns[3],
		Source:    hostname,
		Timestamp: id / int64(time.Millisecond),
		Message:   m.Data().(string),
		Tags:      m.Tags(),
	}

	body, err := msg.CreateProbeEventMsg(event, id, msg.FormatProbeEventMsgp)
	if err != nil {
		log.Printf("Error: unable to convert event to ProbeEventMsgp. %s", err)
		return
	}
	sent := false
	for !sent {
		if err = PostData("events", Token, body); err != nil {
			log.Printf("Error: %s", err)
			time.Sleep(time.Second)
		} else {
			sent = true
		}
	}
}
