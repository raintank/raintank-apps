package client

import (
	"encoding/json"
	"fmt"

	"github.com/raintank/raintank-apps/task-server/model"
)

func (c *Client) GetTasks(q *model.GetTasksQuery) ([]*model.TaskDTO, error) {
	resp, err := c.get("/tasks", q)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err
	}
	tasks := make([]*model.TaskDTO, 0)
	if err := json.Unmarshal(resp.Body, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (c *Client) GetTaskById(id int64) (*model.TaskDTO, error) {
	resp, err := c.get(fmt.Sprintf("/tasks/%d", id), nil)
	if err != nil {
		return nil, err
	}
	if err := resp.Error(); err != nil {
		return nil, err
	}
	task := new(model.TaskDTO)
	if err := json.Unmarshal(resp.Body, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (c *Client) AddTask(t *model.TaskDTO) error {
	resp, err := c.post("/tasks", t)
	if err != nil {
		return err
	}
	if err := resp.Error(); err != nil {
		return err
	}
	if err := json.Unmarshal(resp.Body, t); err != nil {
		return err
	}
	return nil
}

func (c *Client) UpdateTask(t *model.TaskDTO) error {
	resp, err := c.put("/tasks", t)
	if err != nil {
		return err
	}
	if err := resp.Error(); err != nil {
		return err
	}

	if err := json.Unmarshal(resp.Body, t); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteTask(t *model.TaskDTO) error {
	resp, err := c.delete(fmt.Sprintf("/tasks/%d", t.Id), nil)
	if err != nil {
		return err
	}
	if err := resp.Error(); err != nil {
		return err
	}

	return nil
}
