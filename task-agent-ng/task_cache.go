package main

import (
	"encoding/json"

	"github.com/raintank/raintank-apps/task-agent-ng/taskrunner"
	"github.com/raintank/raintank-apps/task-server/model"
	log "github.com/sirupsen/logrus"
)

var taskRunner *taskrunner.TaskRunner

func InitTaskRunner(tsdbgwAddr, tsdbgwAdminAPIKey string) {
	taskRunner = taskrunner.NewTaskRunner(tsdbgwAddr, tsdbgwAdminAPIKey)
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
		taskRunner.UpdateTasks(tasks)
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
		if err := taskRunner.AddTask(&task); err != nil {
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
		if err := taskRunner.AddTask(&task); err != nil {
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
		if err := taskRunner.RemoveTask(&task); err != nil {
			log.Errorf("failed to remove task from cache. %s", err)
		}
	}
}
