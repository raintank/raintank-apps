package ns1

import (
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	. "github.com/intelsdi-x/snap-plugin-utilities/logger"
)

var (
	statusMap = map[string]int{"up": 0, "down": 1}
)

func init() {
	slug.CustomSub = map[string]string{".": "_"}
}

type Ns1 struct {
}

// CollectMetrics collects metrics for testing
func (n *Ns1) CollectMetrics(mts []plugin.Metric) ([]plugin.Metric, error) {
	var err error
	metrics := make([]plugin.Metric, 0)
	apiKey, err := mts[0].Config.GetString("ns1_key")
	if err != nil || apiKey == "" {
		LogError("ns1_key missing from config.")
		return nil, fmt.Errorf("ns1_key missing from config")
	}
	client, err := NewClient("https://api.nsone.net/", apiKey, false)
	if err != nil {
		LogError("failed to create NS1 api client.", "error", err)
		return nil, err
	}
	LogDebug("request to collect metrics", "metric_count", len(mts))
	zoneMts := make([]plugin.Metric, 0)
	monitorMts := make([]plugin.Metric, 0)
	for _, metric := range mts {
		ns := metric.Namespace.Strings()
		if len(ns) > 4 && ns[3] == "zones" {
			zoneMts = append(zoneMts, metric)
		}
		if len(ns) > 4 && ns[3] == "monitoring" {
			monitorMts = append(monitorMts, metric)
		}
	}

	if len(zoneMts) > 0 {
		resp, err := n.ZoneMetrics(client, zoneMts)
		if err != nil {
			LogError("failed to collect metrics.", "error", err)
			return nil, err
		}
		metrics = append(metrics, resp...)
	}
	if len(monitorMts) > 0 {
		resp, err := n.MonitorsMetrics(client, monitorMts)
		if err != nil {
			LogError("failed to collect metrics.", "error", err)
			return nil, err
		}
		metrics = append(metrics, resp...)
	}

	if err != nil {
		LogError("failed to collect metrics.", "error", err)
		return nil, err
	}
	LogDebug("collecting metrics completed", "metric_count", len(metrics))
	return metrics, nil
}

func (n *Ns1) ZoneMetrics(client *Client, mts []plugin.Metric) ([]plugin.Metric, error) {
	metrics := make([]plugin.Metric, 0)
	for _, mt := range mts {
		zone, err := mt.Config.GetString("zone")
		if err != nil || zone == "" {
			LogError("zone missing from config.")
			continue
		}
		zSlug := slug.Make(zone)

		qps, err := client.Qps(zone)
		if err != nil {
			LogError("failed to get zone QPS for zone - "+zone+".", "error", err)
			continue
		}
		ns := mt.Namespace.Strings()
		ns[4] = zSlug
		mt.Namespace = plugin.NewNamespace(ns...)
		mt.Data = qps.Qps
		mt.Timestamp = time.Now()
		metrics = append(metrics, mt)
	}
	return metrics, nil
}

func (n *Ns1) MonitorsMetrics(client *Client, mts []plugin.Metric) ([]plugin.Metric, error) {
	metrics := make([]plugin.Metric, 0)
	ts := time.Now()
	jobs := make(map[string]*MonitoringJob)
	jobsMetrics := make(map[string][]*MonitoringMetric)
	for _, mt := range mts {
		jobId, err := mt.Config.GetString("jobId")
		if err != nil || jobId == "" {
			LogError("jobId missing from config.")
			continue
		}
		jobName, err := mt.Config.GetString("jobName")
		if err != nil || jobName == "" {
			LogError("jobName missing from config.")
			continue
		}
		jSlug := slug.Make(jobName)
		j, ok := jobs[jobId]
		if !ok {
			j, err = client.MonitoringJobById(jobId)
			if err != nil {
				LogError("failed to query for job - "+jobId+" .", "error", err)
				continue
			}
			jobs[jobId] = j
		}

		if mt.Namespace.Element(6).Value == "state" {
			for region, status := range j.Status {
				data, ok := statusMap[status.Status]
				if !ok {
					return nil, fmt.Errorf("Unknown monitor status")
				}
				mt.Data = data
				mt.Timestamp = ts
				ns := mt.Namespace.Strings()
				ns[4] = jSlug
				ns[5] = region
				mt.Namespace = plugin.NewNamespace(ns...)

				metrics = append(metrics, mt)
			}
		} else {
			jobMetrics, ok := jobsMetrics[j.Id]
			if !ok {
				jobMetrics, err := client.MonitoringMetics(j.Id)
				if err != nil {
					LogError("failed to get monitoring metrics for job - "+j.Id, "error", err)
					continue
				}
				jobsMetrics[j.Id] = jobMetrics
			}
			ns := mt.Namespace.Strings()
			for _, jm := range jobMetrics {
				for stat, m := range jm.Metrics {
					if stat != mt.Namespace.Element(6).Value {
						continue
					}
					mt.Data = m.Avg
					mt.Timestamp = ts
					ns[4] = jSlug
					ns[5] = jm.Region
					mt.Namespace = plugin.NewNamespace(ns...)

					metrics = append(metrics, mt)
				}
			}
		}
	}

	return metrics, nil
}

//GetMetricTypes returns metric types for testing
func (n *Ns1) GetMetricTypes(cfg plugin.Config) ([]plugin.Metric, error) {
	mts := []plugin.Metric{}

	mts = append(mts, plugin.Metric{
		Namespace: plugin.NewNamespace("raintank", "apps", "ns1", "zones").
			AddDynamicElement("zone", "DNS Zone Slug").
			AddStaticElement("qps"),
		Version: 1,
	})
	mts = append(mts, plugin.Metric{
		Namespace: plugin.NewNamespace("raintank", "apps", "ns1", "monitoring").
			AddDynamicElement("job", "Monitoring Job Name slug").
			AddDynamicElement("region", "Region").
			AddStaticElement("state"),
		Version: 1,
	})
	mts = append(mts, plugin.Metric{
		Namespace: plugin.NewNamespace("raintank", "apps", "ns1", "monitoring").
			AddDynamicElement("job", "Monitoring Job Name slug").
			AddDynamicElement("region", "Region").
			AddStaticElement("rtt"),
		Version: 1,
	})
	mts = append(mts, plugin.Metric{
		Namespace: plugin.NewNamespace("raintank", "apps", "ns1", "monitoring").
			AddDynamicElement("job", "Monitoring Job Name slug").
			AddDynamicElement("region", "Region").
			AddStaticElement("loss"),
		Version: 1,
	})
	mts = append(mts, plugin.Metric{
		Namespace: plugin.NewNamespace("raintank", "apps", "ns1", "monitoring").
			AddDynamicElement("job", "Monitoring Job Name slug").
			AddDynamicElement("region", "Region").
			AddStaticElement("connect"),
		Version: 1,
	})
	return mts, nil
}

func (f *Ns1) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	policy.AddNewStringRule([]string{"raintank", "apps", "ns1"}, "ns1_key", true)
	policy.AddNewStringRule([]string{"raintank", "apps", "ns1"}, "zone", false)
	policy.AddNewStringRule([]string{"raintank", "apps", "ns1"}, "jobId", false)
	policy.AddNewStringRule([]string{"raintank", "apps", "ns1"}, "jobName", false)
	return *policy, nil
}
