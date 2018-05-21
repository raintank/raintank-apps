// Package taskrunner provides a cron-style execution engine for running arbitrary
// function calls at specified intervals
package taskrunner

import (
	"net/url"
	"sync"

	"github.com/grafana/metrictank/stats"
	"github.com/raintank/raintank-apps/task-agent-ng/collector-ns1/ns1"
	"github.com/raintank/raintank-apps/task-agent-ng/collector-voxter/voxter"
	"github.com/raintank/raintank-apps/task-agent-ng/publisher"
	"github.com/raintank/raintank-apps/task-server/model"
	log "github.com/sirupsen/logrus"
)

var (
	taskAddedCount   = stats.NewCounter32("tasks.added")
	taskUpdatedCount = stats.NewCounter32("tasks.updated")
	taskRemovedCount = stats.NewCounter32("tasks.removed")
	taskInvalidCount = stats.NewCounter32("tasks.invalid")
	taskRunning      = stats.NewGauge32("tasks.running")
)

type Plugin interface {
	CollectMetrics()
}

type nullPlugin struct{}

func (n *nullPlugin) CollectMetrics() {
	return
}

type Task struct {
	Task   *model.TaskDTO
	Ticker *Ticker
	Plugin Plugin
}

func NewTask(task *model.TaskDTO, publisher *publisher.Tsdb) *Task {
	var plugin Plugin
	var err error
	log.Infof("creating task of type %s", task.TaskType)
	switch task.TaskType {
	case "/raintank/apps/ns1":
		plugin, err = ns1.New(task, publisher)
		if err != nil {
			log.Errorf("failed to add ns1 task %d. %s", task.Id, err)
			taskInvalidCount.Inc()
			plugin = new(nullPlugin)
		}
	case "/raintank/apps/voxter":
		plugin, err = voxter.New(task, publisher)
		if err != nil {
			log.Errorf("failed to add ns1 task. %s", err)
			taskInvalidCount.Inc()
			plugin = new(nullPlugin)
		}
	default:
		log.Infof("Unknown Plugin requested. %s", task.TaskType)
		taskInvalidCount.Inc()
		plugin = new(nullPlugin)
	}
	t := &Task{
		Task:   task,
		Ticker: NewTicker(task.Interval, (task.Created.Unix() % task.Interval)),
		Plugin: plugin,
	}
	go t.loop()
	if task.Enabled {
		t.Run()
	}
	return t
}

func (t *Task) loop() {
	log.Infof("Starting execution loop for task %d, Frequency: %d, Offset: %d", t.Task.Id, t.Task.Interval, (t.Task.Created.Unix() % t.Task.Interval))
	for range t.Ticker.C {
		t.Plugin.CollectMetrics()
	}
	log.Infof("execution loop for task %d has ended.", t.Task.Id)
}

func (t *Task) Run() {
	log.Infof("enabling execution thread for task %d", t.Task.Id)
	t.Ticker.Start()
}

func (t *Task) Stop() {
	log.Infof("pausing execution thread for task %d.", t.Task.Id)
	t.Ticker.Stop()
}

func (t *Task) Delete() {
	log.Infof("stopping execution thread for task %d.", t.Task.Id)
	t.Ticker.Delete()
}

type TaskRunner struct {
	sync.RWMutex
	Tasks     map[int64]*Task
	Publisher *publisher.Tsdb
}

func NewTaskRunner(tsdbgwAddr string, tsdbgwApiKey string) *TaskRunner {
	tsdbgwURL, err := url.Parse(tsdbgwAddr)
	if err != nil {
		log.Fatalf("Invalid TSDB url. %s", err)
	}
	return &TaskRunner{
		Publisher: publisher.NewTsdb(tsdbgwURL, tsdbgwApiKey, 1),
		Tasks:     make(map[int64]*Task),
	}
}

// AddTask given a TaskDTO this will create a new job
func (t *TaskRunner) AddTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	return t.addTask(task)
}

func (t *TaskRunner) addTask(task *model.TaskDTO) error {
	if existing, ok := t.Tasks[task.Id]; ok {
		existing.Delete()
		taskRunning.Dec()
	}
	t.Tasks[task.Id] = NewTask(task, t.Publisher)
	taskAddedCount.Inc()
	taskRunning.Inc()
	return nil
}

// UpdateTasks Iterates over the tasks and removes stale entries
func (t *TaskRunner) UpdateTasks(tasks []*model.TaskDTO) {
	seenTaskIds := make(map[int64]struct{})
	t.Lock()
	for _, task := range tasks {
		seenTaskIds[task.Id] = struct{}{}
		existing, ok := t.Tasks[task.Id]
		if !ok {
			// new task
			err := t.addTask(task)
			if err != nil {
				log.Error(err.Error())
			}
		} else if task.Updated.After(existing.Task.Updated) {
			// update task
			err := t.addTask(task)
			if err != nil {
				log.Error(err.Error())
			}
			taskUpdatedCount.Inc()
		}
	}
	tasksToDel := make([]*Task, 0)
	for id, task := range t.Tasks {
		if _, ok := seenTaskIds[id]; !ok {
			tasksToDel = append(tasksToDel, task)
		}
	}
	if len(tasksToDel) > 0 {
		for _, task := range tasksToDel {
			task.Delete()
			delete(t.Tasks, task.Task.Id)
			taskRunning.Dec()
		}
	}
	t.Unlock()
}

func (t *TaskRunner) RemoveTask(task *model.TaskDTO) error {
	t.Lock()
	defer t.Unlock()
	if existing, ok := t.Tasks[task.Id]; ok {
		existing.Delete()
		taskRemovedCount.Inc()
		taskRunning.Dec()
	}
	delete(t.Tasks, task.Id)
	return nil
}
