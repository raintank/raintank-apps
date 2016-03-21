package client

import (
	"encoding/json"

	"github.com/raintank/raintank-apps/task-server/model"
)

func (c *Client) GetMetrics(q *model.GetMetricsQuery) ([]*model.Metric, error) {
	resp, err := c.get("/metrics", q)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err
	}
	metrics := make([]*model.Metric, 0)
	if err := json.Unmarshal(resp.Body, &metrics); err != nil {
		return nil, err
	}
	return metrics, nil
}
