package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana/pkg/log"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/raintank/raintank-apps/task-agent/snap"
	"github.com/raintank/raintank-apps/task-server/model"
)

type TaskCache struct {
	sync.RWMutex
	c         *snap.Client
	Tasks     map[int64]*model.TaskDTO
	SnapTasks map[string]*rbody.ScheduledTask
}

func (t *TaskCache) AddTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	return t.addTask(task)
}

func (t *TaskCache) addTask(task *model.TaskDTO) error {
	t.Tasks[task.Id] = task
	snapTaskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	snapTask, ok := t.SnapTasks[snapTaskName]
	if !ok {
		log.Debug("New task recieved %s", snapTaskName)
		snapTask, err := t.c.CreateSnapTask(task, snapTaskName)
		if err != nil {
			return err
		}
		t.SnapTasks[snapTaskName] = snapTask
	} else {
		log.Debug("task %s already in the cache.", snapTaskName)
		if task.Updated.After(time.Unix(snapTask.CreationTimestamp, 0)) {
			log.Debug("%s needs to be updated", snapTaskName)
			// need to update task.
			if err := t.c.RemoveSnapTask(snapTask); err != nil {
				return err
			}
			snapTask, err := t.c.CreateSnapTask(task, snapTaskName)
			if err != nil {
				return err
			}
			t.SnapTasks[snapTaskName] = snapTask
		}
	}

	return nil
}

func (t *TaskCache) Sync() {
	t.Lock()
	seenTaskIds := make(map[int64]struct{})
	for _, task := range t.Tasks {
		seenTaskIds[task.Id] = struct{}{}
		err := t.addTask(task)
		if err != nil {
			log.Error(3, err.Error())
		}
	}
	tasksToDel := make([]*model.TaskDTO, 0)
	for id, task := range t.Tasks {
		if _, ok := seenTaskIds[id]; !ok {
			tasksToDel = append(tasksToDel, task)
		}
	}
	t.Unlock()
	if len(tasksToDel) > 0 {
		for _, task := range tasksToDel {
			if err := t.RemoveTask(task); err != nil {
				log.Error(3, "Failed to remove task %d", task.Id)
			}
		}
	}
}

func (t *TaskCache) RemoveTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	snapTaskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	snapTask, ok := t.SnapTasks[snapTaskName]
	if !ok {
		log.Debug("task to remove not in cache. %s", snapTaskName)
	} else {
		if err := t.c.RemoveSnapTask(snapTask); err != nil {
			return err
		}
		delete(t.SnapTasks, snapTaskName)
	}
	delete(t.Tasks, task.Id)
	return nil
}

func (t *TaskCache) IndexSnapTasks(tasks []*rbody.ScheduledTask) error {
	t.Lock()
	t.SnapTasks = make(map[string]*rbody.ScheduledTask)
	for _, task := range tasks {
		t.SnapTasks[task.Name] = task
	}
	t.Unlock()
	t.Sync()
	return nil
}

var GlobalTaskCache *TaskCache

func InitTaskCache(snapClient *snap.Client) {
	GlobalTaskCache = &TaskCache{
		c:         snapClient,
		Tasks:     make(map[int64]*model.TaskDTO),
		SnapTasks: make(map[string]*rbody.ScheduledTask),
	}
}

func HandleTaskUpdate() interface{} {
	return func(data []byte) {
		tasks := make([]*model.TaskDTO, 0)
		err := json.Unmarshal(data, &tasks)
		if err != nil {
			log.Error(3, "failed to decode taskUpdate payload. %s", err)
			return
		}
		log.Debug("TaskList. %s", data)
		for _, t := range tasks {
			if err := GlobalTaskCache.AddTask(t); err != nil {
				log.Error(3, "failed to add task to cache. %s", err)
			}
		}
	}
}

func HandleTaskAdd() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Error(3, "failed to decode taskAdd payload. %s", err)
			return
		}
		log.Debug("Adding Task. %s", data)
		if err := GlobalTaskCache.AddTask(&task); err != nil {
			log.Error(3, "failed to add task to cache. %s", err)
		}
	}
}

func HandleTaskRemove() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Error(3, "failed to decode taskAdd payload. %s", err)
			return
		}
		log.Debug("Removing Task. %s", data)
		if err := GlobalTaskCache.RemoveTask(&task); err != nil {
			log.Error(3, "failed to remove task from cache. %s", err)
		}
	}
}
