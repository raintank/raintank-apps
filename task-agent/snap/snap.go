package snap

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/log"
	"github.com/intelsdi-x/snap/core/ctypes"
	"github.com/intelsdi-x/snap/mgmt/rest/client"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/intelsdi-x/snap/scheduler/wmap"
	"github.com/raintank/raintank-apps/task-server/model"
)

type Client struct {
	NodeName    string
	TsdbAddr    string
	ApiKey      string
	c           *client.Client
	url         *url.URL
	connected   bool
	ConnectChan chan struct{}
}

func NewClient(nodeName, tsdbAddr, apiKey string, u *url.URL) (*Client, error) {
	addr := strings.TrimSuffix(u.String(), "/")
	c, err := client.New(addr, "v1", false)
	if err != nil {
		return nil, err
	}
	return &Client{
		NodeName:    nodeName,
		TsdbAddr:    tsdbAddr,
		ApiKey:      apiKey,
		c:           c,
		url:         u,
		connected:   false,
		ConnectChan: make(chan struct{}),
	}, nil
}

func (c *Client) Run() {
	log.Info("running SnapClient supervisor.")
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		conf, err := c.GetSnapGlobalConfig()
		if err != nil {
			log.Debug("Snap server is unreachable. %s", err.Error())
			if c.connected {
				log.Error(3, "Snap server unreachable. %s", err.Error())
				c.connected = false
			}
			continue
		}
		if !c.connected {
			log.Info("connected to snap server.")
			c.connected = true
		}
		if _, ok := conf.Table()["raintank_agent_name"]; !ok {
			err := c.SetSnapGlobalConfig()
			if err != nil {
				continue
			}
			c.ConnectChan <- struct{}{}
		}
	}
}

func (c *Client) GetSnapMetrics() ([]*rbody.Metric, error) {
	resp := c.c.GetMetricCatalog()
	return resp.Catalog, resp.Err
}

func (c *Client) GetSnapTasks() ([]*rbody.ScheduledTask, error) {
	resp := c.c.GetTasks()
	var tasks []*rbody.ScheduledTask
	if resp.Err == nil {
		tasks = make([]*rbody.ScheduledTask, len(resp.ScheduledTasks))
		for i, t := range resp.ScheduledTasks {
			tasks[i] = &t
		}
	}
	return tasks, resp.Err
}

func (c *Client) RemoveSnapTask(task *rbody.ScheduledTask) error {
	stopResp := c.c.StopTask(task.ID)
	if stopResp.Err != nil {
		return stopResp.Err
	}
	removeResp := c.c.RemoveTask(task.ID)
	return removeResp.Err
}

func (c *Client) CreateSnapTask(t *model.TaskDTO, name string) (*rbody.ScheduledTask, error) {
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

	resp := c.c.CreateTask(s, wf, name, "10s", true)
	log.Debug("%v", resp)
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

func (c *Client) GetSnapGlobalConfig() (*rbody.PluginConfigItem, error) {
	resp := c.c.GetPluginConfig("", "", "")
	return resp.PluginConfigItem, resp.Err

}

func (c *Client) SetSnapGlobalConfig() error {
	agentName := ctypes.ConfigValueStr{
		Value: c.NodeName,
	}
	resp := c.c.SetPluginConfig("", "", "", "raintank_agent_name", agentName)
	if resp.Err != nil {
		return resp.Err
	}

	url, err := url.Parse(c.TsdbAddr)
	if err != nil {
		return err
	}
	tsdbUrl := ctypes.ConfigValueStr{
		Value: url.String(),
	}
	resp = c.c.SetPluginConfig("", "", "", "raintank_tsdb_url", tsdbUrl)
	if resp.Err != nil {
		return resp.Err
	}

	key := ctypes.ConfigValueStr{
		Value: c.ApiKey,
	}
	resp = c.c.SetPluginConfig("", "", "", "raintank_api_key", key)
	if resp.Err != nil {
		return resp.Err
	}
	return nil
}
