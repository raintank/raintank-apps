package client

import (
	"encoding/json"
	"fmt"

	"github.com/raintank/raintank-apps/task-server/model"
)

func (c *Client) GetAgents(q *model.GetAgentsQuery) ([]*model.AgentDTO, error) {
	resp, err := c.get("/agents", q)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err
	}
	agents := make([]*model.AgentDTO, 0)
	if err := json.Unmarshal(resp.Body, &agents); err != nil {
		return nil, err
	}
	return agents, nil
}

func (c *Client) GetAgentById(id int64) (*model.AgentDTO, error) {
	resp, err := c.get(fmt.Sprintf("/agents/%d", id), nil)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err
	}
	agent := new(model.AgentDTO)
	if err := json.Unmarshal(resp.Body, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

func (c *Client) AddAgent(a *model.AgentDTO) error {
	resp, err := c.post("/agents", a)
	if err != nil {
		return err
	}
	if err := resp.Error(); err != nil {
		return err
	}
	if err := json.Unmarshal(resp.Body, a); err != nil {
		return err
	}
	return nil
}

func (c *Client) UpdateAgent(a *model.AgentDTO) error {
	resp, err := c.put("/agents", a)
	if err != nil {
		return err
	}
	if err := resp.Error(); err != nil {
		return err
	}

	if err := json.Unmarshal(resp.Body, a); err != nil {
		return err
	}
	return nil
}
