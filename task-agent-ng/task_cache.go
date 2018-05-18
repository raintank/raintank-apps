package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/grafana/metrictank/stats"
	"github.com/raintank/raintank-apps/task-agent-ng/collector-ns1/ns1"
	"github.com/raintank/raintank-apps/task-agent-ng/collector-voxter/voxter"
	"github.com/raintank/raintank-apps/task-agent-ng/publisher"

	"github.com/raintank/raintank-apps/task-agent-ng/taskrunner"
	"github.com/raintank/raintank-apps/task-server/model"
	log "github.com/sirupsen/logrus"
)

var (
	taskAddedCount   = stats.NewCounter32("tasks.added")
	taskUpdatedCount = stats.NewCounter32("tasks.updated")
	taskRemovedCount = stats.NewCounter32("tasks.removed")
	taskInvalidCount = stats.NewCounter32("tasks.invalid")
)

type Plugin interface {
	CollectMetrics()
}

type TaskCache struct {
	sync.RWMutex
	tsdbgwURL    *url.URL
	tsdbgwAPIKey *string
	Tasks        map[int64]*model.TaskDTO
	initialized  bool
	TaskRunner   taskrunner.TaskRunner
	Publisher    *publisher.Tsdb
}

// AddTask given a TaskDTO this will create a new job
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
	taskName := fmt.Sprintf("raintank-apps:%d", task.Id)
	// check if job already exists

	aJob, ok := t.TaskRunner.Exists(int(task.Id))
	if !ok {
		log.Debugf("New task received %s", taskName)
		t.AddToTaskRunner(task)
	} else {
		log.Debugf("task %s already in the cache.", taskName)
		// check age of creation timestamp vs the task updated Timestamp
		// if the running job is has an older timestamp, remove it and create a new job
		if task.Updated.After(time.Unix(aJob.CreationTimestamp, 0)) {
			log.Infof("%s is stale and needs to be updated", taskName)
			// need to update task
			// remove it first
			t.TaskRunner.Remove(aJob)
			// then add it
			t.AddToTaskRunner(task)
		}
	}

	return nil
}

// AddToTaskRunner decodes the type of plugin to use from the config and creates a new cron task
func (t *TaskCache) AddToTaskRunner(task *model.TaskDTO) {
	var plugin Plugin
	var err error
	log.Infof("adding task of type %s", task.TaskType)
	switch task.TaskType {
	case "/raintank/apps/ns1":
		plugin, err = ns1.New(task, t.Publisher)
		if err != nil {
			log.Errorf("failed to add ns1 task %d. %s", task.Id, err)
			taskInvalidCount.Inc()
			return
		}
	case "/raintank/apps/voxter":
		plugin, err = voxter.New(task, t.Publisher)
		if err != nil {
			log.Errorf("failed to add ns1 task. %s", err)
			taskInvalidCount.Inc()
			return
		}
		return
	default:
		log.Info("Unknown Plugin Needed!")
		return
	}
	taskAddedCount.Inc()
	sched := fmt.Sprintf("@every %ds", task.Interval)
	id1 := t.TaskRunner.Add(int(task.Id), sched, plugin.CollectMetrics)
	log.Infof("task %d assigned cron id: %v", task.Id, id1)
}

// UpdateTasks Iterates over the tasks and removes stale entries
func (t *TaskCache) UpdateTasks(tasks []*model.TaskDTO) {
	seenTaskIds := make(map[int64]struct{})
	t.Lock()
	for _, task := range tasks {
		seenTaskIds[task.Id] = struct{}{}
		err := t.addTask(task)
		if err != nil {
			log.Error(err.Error())
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
				log.Errorf("Failed to remove task %d", task.Id)
			}
		}
	}
}

func (t *TaskCache) RemoveTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	log.Debugf("removing task %d", task.Id)
	if err := t.removeActiveTask(task); err != nil {
		return err
	}
	delete(t.Tasks, task.Id)
	return nil
}

func (t *TaskCache) removeActiveTask(task *model.TaskDTO) error {
	aJob, ok := t.TaskRunner.Exists(int(task.Id))
	if !ok {
		log.Debugf("removeActiveTask: task does not exist %d", task.Id)
		return fmt.Errorf("task does not exist")
	}
	t.TaskRunner.Remove(aJob)
	taskRemovedCount.Inc()
	return nil
}

var GlobalTaskCache *TaskCache

func InitTaskCache(tsdbgwAddr *string, tsdbgwApiKey *string) {
	log.Infof("TSDB-GW URL is %s", *tsdbgwAddr)
	tsdbgwURL, err := url.Parse(*tsdbgwAddr)
	if err != nil {
		log.Fatalf("Invalid TSDB url. %s", err)
	}
	GlobalTaskCache = &TaskCache{
		tsdbgwURL: tsdbgwURL,
		Tasks:     make(map[int64]*model.TaskDTO),
	}

	GlobalTaskCache.Publisher = publisher.NewTsdb(tsdbgwURL, *tsdbgwApiKey, 1)
	GlobalTaskCache.TaskRunner = taskrunner.TaskRunner{}
	GlobalTaskCache.TaskRunner.Init()
	GlobalTaskCache.initialized = true
}

func HandleTaskList() interface{} {
	return func(data []byte) {
		tasks := make([]*model.TaskDTO, 0)
		err := json.Unmarshal(data, &tasks)
		if err != nil {
			log.Errorf("failed to decode taskUpdate payload. %s", err)
			return
		}
		log.Debugf("TaskList. %s", data)
		GlobalTaskCache.UpdateTasks(tasks)
	}
}

func HandleTaskUpdate() interface{} {
	return func(data []byte) {
		task := model.TaskDTO{}
		err := json.Unmarshal(data, &task)
		if err != nil {
			log.Errorf("failed to decode taskUpdate payload. %s", err)
			return
		}
		log.Debugf("TaskUpdate. %s", data)
		if err := GlobalTaskCache.AddTask(&task); err != nil {
			log.Errorf("failed to add task to cache. %s", err)
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
		log.Debug("Adding Task. %s", data)
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
