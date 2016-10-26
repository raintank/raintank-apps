package hostedtsdb

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	"gopkg.in/raintank/schema.v1"
	"gopkg.in/raintank/schema.v1/msg"
)

const (
	maxMetricsPerPayload = 3000
)

var (
	RemoteUrl *url.URL
	Token     string
	log       *logrus.Entry
)

func init() {
	log = logrus.WithFields(logrus.Fields{
		"plugin-name":    "rt-hostedtsdb",
		"plugin-version": 2,
		"plugin-type":    "publisher",
	})

	logrus.SetLevel(logrus.DebugLevel)
}

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
	log.Debug("writing %d metrics to API", len(metrics))
	id := time.Now().UnixNano()
	body, err := msg.CreateMsg(metrics, id, msg.FormatMetricDataArrayMsgp)
	if err != nil {
		log.Errorf("Unable to convert metrics to MetricDataArrayMsgp. %s", err)
		return
	}
	sent := false
	for !sent {
		if err = PostData("metrics", Token, body); err != nil {
			log.Errorf("failed to post metrics. %s", err)
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

func (f *HostedtsdbPublisher) Publish(metrics []plugin.Metric, cfg plugin.Config) error {
	log.Debug("publishing %d metrics to %v", len(metrics), cfg)

	// set the RemoteURL and Token when the first metrics is recieved.
	var err error
	if RemoteUrl == nil {
		remote, err := cfg.GetString("raintank_tsdb_url")
		if err != nil {
			return err
		}
		if !strings.HasSuffix(remote, "/") {
			remote += "/"
		}
		RemoteUrl, err = url.Parse(remote)
		if err != nil {
			return err
		}

	}
	if Token == "" {
		Token, err = cfg.GetString("raintank_api_key")
		if err != nil {
			return err
		}
	}
	//-----------------

	interval, err := cfg.GetInt("interval")
	if err != nil {
		return err
	}
	orgId, err := cfg.GetInt("orgId")
	if err != nil {
		return err
	}

	metricsArray := make([]*schema.MetricData, len(metrics))
	for i, m := range metrics {
		var value float64
		switch m.Data.(type) {
		case string:
			//payload is an event.
			go sendEvent(int64(orgId), &m)
			continue
		case int:
			value = float64(m.Data.(int))
		case int8:
			value = float64(m.Data.(int8))
		case int16:
			value = float64(m.Data.(int16))
		case int32:
			value = float64(m.Data.(int32))
		case int64:
			value = float64(m.Data.(int64))
		case uint8:
			value = float64(m.Data.(uint8))
		case uint16:
			value = float64(m.Data.(uint16))
		case uint32:
			value = float64(m.Data.(uint32))
		case uint64:
			value = float64(m.Data.(uint64))
		case float32:
			value = float64(m.Data.(float32))
		case float64:
			value = m.Data.(float64)
		default:
			return errors.New("unknown data type")
		}

		tags := make([]string, 0)
		mtype := "gauge"
		unit := ""
		for k, v := range m.Tags {
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
			OrgId:    int(orgId),
			Name:     m.Namespace.Key(),
			Metric:   m.Namespace.Key(),
			Interval: int(interval),
			Value:    value,
			Time:     m.Timestamp.Unix(),
			Mtype:    mtype,
			Unit:     unit,
			Tags:     tags,
		}
		metricsArray[i].SetId()
	}
	writeQueue.Add(metricsArray)

	return nil
}

func (f *HostedtsdbPublisher) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	policy.AddNewStringRule([]string{""}, "raintank_tsdb_url", true)
	policy.AddNewStringRule([]string{""}, "raintank_api_key", true)
	policy.AddNewIntRule([]string{""}, "interval", true)
	policy.AddNewIntRule([]string{""}, "orgId", true)
	return *policy, nil
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

func sendEvent(orgId int64, m *plugin.Metric) {
	ns := m.Namespace.Strings()
	if len(ns) != 4 {
		log.Error("Invalid event metric. Expected namesapce to be 4 fields.")
		return
	}
	if ns[0] != "worldping" || ns[1] != "event" {
		log.Error("Invalid event metrics.  Metrics hould begin with 'worldping.event'")
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
		Message:   m.Data.(string),
		Tags:      m.Tags,
	}

	body, err := msg.CreateProbeEventMsg(event, id, msg.FormatProbeEventMsgp)
	if err != nil {
		log.Errorf("unable to convert event to ProbeEventMsgp. %s", err)
		return
	}
	sent := false
	for !sent {
		if err = PostData("events", Token, body); err != nil {
			log.Errorf("filed to POST event payload: %s", err)
			time.Sleep(time.Second)
		} else {
			sent = true
		}
	}
}
