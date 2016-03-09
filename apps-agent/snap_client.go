package main

import (
	"fmt"
	"net/url"

	"github.com/intelsdi-x/snap/core/ctypes"
	"github.com/intelsdi-x/snap/mgmt/rest/client"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/intelsdi-x/snap/scheduler/wmap"
	"github.com/raintank/raintank-apps/apps-server/model"
)

var SnapClient *client.Client

func InitSnapClient(u *url.URL) {
	SnapClient, _ = client.New(u.String(), "v1", false)
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
	token := ""
	for ns, conf := range t.Config {
		for key, value := range conf {
			wf.CollectNode.AddConfigItem(ns, key, value)
			if key == "token" {
				token = value.(string)
			}
		}
	}
	publisher := getPublisher(
		1, //TODO: replace with actual orgId
		t.Interval,
		token,
	)
	if err := wf.CollectNode.Add(publisher); err != nil {
		return nil, err
	}

	resp := SnapClient.CreateTask(s, wf, name, "10s", true)
	log.Debugf("%v", resp)
	var newTask rbody.ScheduledTask
	if resp.Err == nil {
		newTask = rbody.ScheduledTask(*resp.AddScheduledTask)
	}
	return &newTask, resp.Err
}

func getPublisher(orgId, interval int64, token string) *wmap.PublishWorkflowMapNode {
	return &wmap.PublishWorkflowMapNode{
		Name: "rt-hostedtsdb",
		Config: map[string]interface{}{
			"interval": interval,
			"orgId":    orgId,
		},
	}
}

func SetSnapGlobalConfig() error {
	agentName := ctypes.ConfigValueStr{
		Value: *nodeName,
	}
	resp := SnapClient.SetPluginConfig("", "", "", "raintank_agent_name", agentName)
	if resp.Err != nil {
		return resp.Err
	}

	url, err := url.Parse(*tsdbAddr)
	if err != nil {
		return err
	}
	tsdbUrl := ctypes.ConfigValueStr{
		Value: url.String(),
	}
	resp = SnapClient.SetPluginConfig("", "", "", "raintank_tsdb_url", tsdbUrl)
	if resp.Err != nil {
		return resp.Err
	}

	key := ctypes.ConfigValueStr{
		Value: *apiKey,
	}
	resp = SnapClient.SetPluginConfig("", "", "", "raintank_api_key", key)
	if resp.Err != nil {
		return resp.Err
	}
	return nil
}
