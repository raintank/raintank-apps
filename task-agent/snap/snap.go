package snap

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/intelsdi-x/snap/core/ctypes"
	"github.com/intelsdi-x/snap/mgmt/rest/client"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/intelsdi-x/snap/scheduler/wmap"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/worldping-api/pkg/log"
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
	go c.watchSnapServer()
	go c.watchTasks()
}

func (c *Client) watchSnapServer() {
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
		if _, ok := conf.Table()["raintank_agent_name"]; !ok {
			err := c.SetSnapGlobalConfig()
			if err != nil {
				continue
			}
		}
		if !c.connected {
			log.Info("connected to snap server.")
			c.connected = true
			c.ConnectChan <- struct{}{}
		}
	}
}

func (c *Client) watchTasks() {
	log.Info("running SnapClient task supervisor.")
	ticker := time.NewTicker(time.Minute)

	for range ticker.C {
		tasks, err := c.GetSnapTasks()
		if err != nil {
			log.Error(3, "Failed get task list from snap server.")
		}
		syncNeeded := false
		for _, t := range tasks {
			if t.State == "Disabled" {
				log.Info("task %s is marked as disabled. Removing it.", t.Name)
				err = c.RemoveSnapTask(t)
				if err != nil {
					log.Error(3, "Failed to remove task. %s", err)
				}
				syncNeeded = true
			}
		}
		if syncNeeded {
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
	resp := c.c.GetTask(task.ID)
	if resp.Err != nil {
		return resp.Err
	}
	if resp.State == "Disabled" {
		// need to enable the task before stopping it.
		enableResp := c.c.EnableTask(task.ID)
		if enableResp.Err != nil {
			return enableResp.Err
		}
		resp = c.c.GetTask(task.ID)
		if resp.Err != nil {
			return resp.Err
		}
	}
	if resp.State != "Stopped" {
		stopResp := c.c.StopTask(task.ID)
		if stopResp.Err != nil {
			return stopResp.Err
		}
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
		t.OrgId,
		t.Interval,
		token,
	)
	if err := wf.CollectNode.Add(publisher); err != nil {
		return nil, err
	}

	resp := c.c.CreateTask(s, wf, name, "300s", true, 100)
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
