package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/raintank/raintank-apps/task-agent/snap"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/worldping-api/pkg/log"
)

type TaskCache struct {
	sync.RWMutex
	c           *snap.Client
	Tasks       map[int64]*model.TaskDTO
	SnapTasks   map[string]*rbody.ScheduledTask
	initialized bool
}

func (t *TaskCache) AddTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	return t.addTask(task)
}

func (t *TaskCache) addTask(task *model.TaskDTO) error {
	t.Tasks[task.Id] = task
	if !t.initialized {
		return nil
	}
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

func (t *TaskCache) UpdateTasks(tasks []*model.TaskDTO) {
	seenTaskIds := make(map[int64]struct{})
	t.Lock()
	for _, task := range tasks {
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

func (t *TaskCache) Sync() {
	tasksByName := make(map[string]*model.TaskDTO)
	t.Lock()
	for _, task := range t.Tasks {
		name := fmt.Sprintf("raintank-apps:%d", task.Id)
		tasksByName[name] = task
		log.Debug("seen %s", name)
		err := t.addTask(task)
		if err != nil {
			log.Error(3, err.Error())
		}
	}

	for name := range t.SnapTasks {
		// dont remove tasks that were not added by us.
		if !strings.HasPrefix(name, "raintank-apps") {
			continue
		}
		if _, ok := tasksByName[name]; !ok {
			log.Info("%s not in taskList. removing from snap.", name)
			if err := t.removeSnapTask(name); err != nil {
				log.Error(3, "failed to remove snapTask %s. %s", name, err)
			}
		}
	}
	t.Unlock()

}

func (t *TaskCache) RemoveTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	snapTaskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	log.Debug("removing snap task %s", snapTaskName)
	if err := t.removeSnapTask(snapTaskName); err != nil {
		return err
	}

	delete(t.Tasks, task.Id)
	return nil
}

func (t *TaskCache) removeSnapTask(taskName string) error {
	snapTask, ok := t.SnapTasks[taskName]
	if !ok {
		log.Debug("task to remove not in cache. %s", taskName)
	} else {
		if err := t.c.RemoveSnapTask(snapTask); err != nil {
			return err
		}
		delete(t.SnapTasks, taskName)
	}

	return nil
}

func (t *TaskCache) IndexSnapTasks() error {
	log.Debug("running indexSnapTasks")
	tasks, err := t.c.GetSnapTasks()
	if err != nil {
		return err
	}
	t.Lock()
	t.SnapTasks = make(map[string]*rbody.ScheduledTask)
	for _, task := range tasks {
		log.Debug("task %s running in snap.", task.Name)
		t.SnapTasks[task.Name] = task
	}
	if !t.initialized {
		t.initialized = true
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

func HandleTaskList() interface{} {
	return func(data []byte) {
		tasks := make([]*model.TaskDTO, 0)
		err := json.Unmarshal(data, &tasks)
		if err != nil {
			log.Error(3, "failed to decode taskUpdate payload. %s", err)
			return
		}
		log.Debug("TaskList. %s", data)
		GlobalTaskCache.UpdateTasks(tasks)
	}
}

func HandleTaskUpdate() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Error(3, "failed to decode taskUpdate payload. %s", err)
			return
		}
		log.Debug("TaskUpdate. %s", data)
		if err := GlobalTaskCache.AddTask(&task); err != nil {
			log.Error(3, "failed to add task to cache. %s", err)
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
