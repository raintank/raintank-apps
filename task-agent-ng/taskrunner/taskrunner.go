// Package taskrunner provides a cron-style execution engine for running arbitrary
// function calls at specified intervals
package taskrunner

import (
	"fmt"
	"time"

	"github.com/grafana/metrictank/stats"
	log "github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
)

type RTAMetric struct {
	Name       string
	MetricName string
	Value      float64
	Tags       map[string]string
	Timestamp  int64
	Unit       string
	Zone       string
}

var (
	runnerAddedCount   = stats.NewCounter32("runner.tasks.added")
	runnerRemovedCount = stats.NewCounter32("runner.tasks.removed")
	runnerInitialized  = stats.NewGauge32("runner.initialized")
	runnerActiveTasks  = stats.NewGauge32("runner.tasks.active")
)

// TaskMeta holds creation and execution data about a collector
type TaskMeta struct {
	ID                 int
	Name               string
	CreationTimestamp  int64
	LastRunTimestamp   int64
	FailedCount        int64
	LastFailureMessage string
	State              int
	CronID             cron.EntryID
}

// TaskRunner holds the cron object and all job Task Meta information
type TaskRunner struct {
	Master *cron.Cron
	Jobs   map[int]TaskMeta
}

// Init required to initialize cron and tracking map
func (c *TaskRunner) Init() {
	log.Info("Creating Cron")
	c.Jobs = make(map[int]TaskMeta)
	c.Master = cron.New()
	c.Master.Start()
	runnerInitialized.Inc()
}

// Add Creates a new cron job to execute an arbitrary collection
func (c *TaskRunner) Add(taskID int, jobSchedule string, jobFunc func()) cron.EntryID {
	log.Infof("Adding job for task %d to Cron", taskID)
	taskName := fmt.Sprintf("raintank-apps:%d", taskID)

	jobID, _ := c.Master.AddFunc(jobSchedule, jobFunc)
	runnerAddedCount.Inc()
	runnerActiveTasks.Set(len(c.Master.Entries()))
	c.Jobs[int(taskID)] = TaskMeta{
		CronID:            jobID,
		ID:                taskID,
		CreationTimestamp: time.Now().Unix(),
		Name:              taskName,
		State:             1,
	}
	return jobID
}

// Remove terminates a runnning job and removes from map
func (c *TaskRunner) Remove(aJob TaskMeta) {
	// TODO use task meta vs the cron id
	log.Infof("Removing job with cronId %d from cron", aJob.CronID)
	c.Master.Remove(aJob.CronID)
	log.Infof("Removing job wit ID %d from job tracker", aJob.ID)
	delete(c.Jobs, aJob.ID)
	runnerRemovedCount.Inc()
	runnerActiveTasks.Set(len(c.Master.Entries()))
}

// Exists returns the job metadata and true if it is found
func (c *TaskRunner) Exists(jobID int) (TaskMeta, bool) {
	log.Debugf("Checking for job %d", jobID)
	aJob, ok := c.Jobs[jobID]
	if ok {
		// check if the id matches
		if aJob.ID == jobID {
			return aJob, true
		}
	}
	return aJob, false
}

// Shutdown stops the cron service
func (c *TaskRunner) Shutdown() {
	log.Info("Shutting down...")
	defer c.Master.Stop()
	runnerInitialized.Set(0)
}
