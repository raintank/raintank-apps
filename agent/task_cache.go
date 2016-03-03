package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/raintank/raintank-apps/pkg/session"
	"github.com/raintank/raintank-apps/server/model"
)

type TaskCache struct {
	sync.RWMutex
	Tasks     map[int64]*model.TaskDTO
	SnapTasks map[string]*rbody.ScheduledTask
}

func (t *TaskCache) AddTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	snapTaskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	snapTask, ok := t.SnapTasks[snapTaskName]
	if !ok {
		log.Debugf("New task recieved %s", snapTaskName)
		snapTask, err := CreateSnapTask(task, snapTaskName)
		if err != nil {
			return err
		}
		t.SnapTasks[snapTaskName] = snapTask
	} else {
		log.Debugf("task %s already in the cache.", snapTaskName)
		if task.Updated.After(time.Unix(snapTask.CreationTimestamp, 0)) {
			log.Debugf("%s needs to be updated", snapTaskName)
			// need to update task.
			if err := RemoveSnapTask(snapTask); err != nil {
				return err
			}
			snapTask, err := CreateSnapTask(task, snapTaskName)
			if err != nil {
				return err
			}
			t.SnapTasks[snapTaskName] = snapTask
		}
	}
	t.Tasks[task.Id] = task
	return nil
}

func (t *TaskCache) RemoveTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	snapTaskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	snapTask, ok := t.SnapTasks[snapTaskName]
	if !ok {
		log.Debugf("task to remove not in cache. %s", snapTaskName)
	} else {
		if err := RemoveSnapTask(snapTask); err != nil {
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
	return nil
}

var GlobalTaskCache *TaskCache

func init() {
	GlobalTaskCache = &TaskCache{
		Tasks:     make(map[int64]*model.TaskDTO),
		SnapTasks: make(map[string]*rbody.ScheduledTask),
	}
}

func HandleTaskUpdate() interface{} {
	return func(data []byte) {
		tasks := make([]*model.TaskDTO, 0)
		err := json.Unmarshal(data, &tasks)
		if err != nil {
			log.Errorf("failed to decode taskUpdate payload. %s", err)
			return
		}
		log.Debugf("TaskList. %s", data)
		for _, t := range tasks {
			if err := GlobalTaskCache.AddTask(t); err != nil {
				log.Errorf("failed to add task to cache. %s", err)
			}
		}
	}
}

func HandleTaskAdd() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Errorf("failed to decode taskAdd payload. %s", err)
			return
		}
		log.Debugf("Adding Task. %s", data)
		if err := GlobalTaskCache.AddTask(&task); err != nil {
			log.Errorf("failed to add task to cache. %s", err)
		}
	}
}

func HandleTaskRemove() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Errorf("failed to decode taskAdd payload. %s", err)
			return
		}
		log.Debugf("Removing Task. %s", data)
		if err := GlobalTaskCache.RemoveTask(&task); err != nil {
			log.Errorf("failed to remove task from cache. %s", err)
		}
	}
}

func IndexTasks(sess *session.Session, shutdownStart chan struct{}) {
	ticker := time.NewTicker(time.Minute * 5)
	for {
		select {
		case <-shutdownStart:
			return
		case <-ticker.C:
			taskList, err := GetSnapTasks()
			if err != nil {
				log.Error(err)
				continue
			}
			if err := GlobalTaskCache.IndexSnapTasks(taskList); err != nil {
				log.Errorf("failed to add task to cache. %s", err)
			}
		}
	}
}
