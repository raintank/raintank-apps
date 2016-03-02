package main

import (
	"fmt"
	"net/url"

	"github.com/intelsdi-x/snap/mgmt/rest/client"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/intelsdi-x/snap/scheduler/wmap"
	"github.com/raintank/raintank-apps/server/model"
)

var SnapClient *client.Client

var DefaultPublisher = &wmap.PublishWorkflowMapNode{
	Name: "file",
	Config: map[string]interface{}{
		"file": "/tmp/gitstats.out",
	},
}

func InitSnapClient(u *url.URL) {
	SnapClient = client.New(u.String(), "v1", false)
}

func GetSnapMetrics() ([]*rbody.Metric, error) {
	resp := SnapClient.GetMetricCatalog()
	return resp.Catalog, resp.Err
}

func GetSnapTasks() ([]*rbody.ScheduledTask, error) {
	resp := SnapClient.GetTasks()
	var tasks []*rbody.ScheduledTask
	if resp.Err == nil {
		tasks = make([]*rbody.ScheduledTask, len(resp.ScheduledTasks))
		for i, t := range resp.ScheduledTasks {
			tasks[i] = &t
		}
	}
	return tasks, resp.Err
}

func RemoveSnapTask(task *rbody.ScheduledTask) error {
	stopResp := SnapClient.StopTask(task.ID)
	if stopResp.Err != nil {
		return stopResp.Err
	}
	removeResp := SnapClient.RemoveTask(task.ID)
	return removeResp.Err
}

func CreateSnapTask(t *model.TaskDTO, name string) (*rbody.ScheduledTask, error) {
	s := &client.Schedule{
		Type:     "simple",
		Interval: fmt.Sprintf("%ds", t.Interval),
	}
	wf := wmap.NewWorkflowMap()
	for ns, ver := range t.Metrics {
		if err := wf.CollectNode.AddMetric(ns, int(ver)); err != nil {
			return nil, err
		}
	}
	for ns, conf := range t.Config {
		for key, value := range conf {
			wf.CollectNode.AddConfigItem(ns, key, value)
		}
	}

	if err := wf.CollectNode.Add(DefaultPublisher); err != nil {
		return nil, err
	}

	resp := SnapClient.CreateTask(s, wf, name, "10s", true)
	newTask := rbody.ScheduledTask(*resp.AddScheduledTask)
	return &newTask, resp.Err
}
