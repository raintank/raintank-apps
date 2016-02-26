package sqlstore

import (
	"strconv"
	"time"

	"github.com/raintank/raintank-apps/server/model"
)

type taskWithMetric struct {
	model.Task `xorm:"extends"`
	Namespace  string
	Version    int64
}

type taskWithMetrics []*taskWithMetric

func (taskWithMetrics) TableName() string {
	return "task"
}

func (rows taskWithMetrics) ToTaskDTO() []*model.TaskDTO {
	taskById := make(map[int64]*model.TaskDTO)
	for _, r := range rows {
		t, ok := taskById[r.Id]
		if !ok {
			taskById[r.Id] = &model.TaskDTO{
				Id:       r.Id,
				Owner:    r.Owner,
				Name:     r.Name,
				Enabled:  r.Enabled,
				Interval: r.Interval,
				Route:    r.Route,
				Config:   r.Config,
				Created:  r.Created,
				Updated:  r.Updated,
				Metrics:  map[string]int64{r.Namespace: r.Version},
			}
		} else {
			t.Metrics[r.Namespace] = r.Version
		}
	}
	tasks := make([]*model.TaskDTO, len(taskById))
	i := 0
	for _, t := range taskById {
		tasks[i] = t
		i++
	}
	return tasks
}

func GetTasks(query *model.GetTasksQuery) ([]*model.TaskDTO, error) {
	sess, err := newSession(false, "task")
	if err != nil {
		return nil, err
	}
	return getTasks(sess, query)
}

func getTasks(sess *session, query *model.GetTasksQuery) ([]*model.TaskDTO, error) {
	var t taskWithMetrics
	if query.Owner != "" {
		sess.Where("task.owner = ?", query.Owner)
	}
	if query.Enabled != "" {
		enabled, err := strconv.ParseBool(query.Enabled)
		if err != nil {
			return nil, err
		}
		sess.Where("task.enabled=?", enabled)
	}

	if query.Metric != "" {
		sess.Join("INNER", []string{"task_metric", "tm"}, "task.id = at.tm.task_id").
			Where("tm.namespace=?", query.Metric)
		if query.MetricVersion == 0 {
			// get the latest version.
			sess.And("tm.version = (SELECT MAX(version) FROM metric WHERE namespace=? group by version)", query.Metric)
		} else {
			sess.And("tm.version=?", query.MetricVersion)
		}
	}
	sess.Cols(
		"task.id",
		"task.name",
		"task.owner",
		"task.enabled",
		"task.interval",
		"task.config",
		"task.route",
		"task.created",
		"task.updated",
		"task_metric.namespace",
		"task_metric.version",
	)
	err := sess.Join("LEFT", "task_metric", "task.id = task_metric.task_id").Find(&t)
	if err != nil {
		return nil, err
	}
	return t.ToTaskDTO(), nil
}

func GetTaskById(id int64, owner string) (*model.TaskDTO, error) {
	sess, err := newSession(false, "task")
	if err != nil {
		return nil, err
	}
	return getTaskById(sess, id, owner)
}

func getTaskById(sess *session, id int64, owner string) (*model.TaskDTO, error) {
	var t taskWithMetrics
	err := sess.Where("task.id=? AND owner=?", id, owner).Join("INNER", "task_metric", "task.id = task_metric.task_id").Find(&t)
	if err != nil {
		return nil, err
	}
	if len(t) == 0 {
		return nil, nil
	}
	return t.ToTaskDTO()[0], nil
}

func AddTask(t *model.TaskDTO) error {
	sess, err := newSession(true, "task")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = addTask(sess, t); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func addTask(sess *session, t *model.TaskDTO) error {
	task := model.Task{
		Name:     t.Name,
		Owner:    t.Owner,
		Interval: t.Interval,
		Enabled:  t.Enabled,
		Config:   t.Config,
		Route:    t.Route,
		Created:  time.Now(),
		Updated:  time.Now(),
	}
	sess.UseBool("enabled")
	if _, err := sess.Insert(&task); err != nil {
		log.Debugf("could not insert into task. %s", err)
		return err
	}
	t.Created = task.Created
	t.Updated = task.Updated
	t.Id = task.Id

	// handle metrics.
	metrics := make([]*model.TaskMetric, 0, len(t.Metrics))
	for namespace, ver := range t.Metrics {
		//TODO: validate each Metric.
		metrics = append(metrics, &model.TaskMetric{
			TaskId:    t.Id,
			Namespace: namespace,
			Version:   ver,
			Created:   time.Now(),
		})
	}
	if len(metrics) > 0 {
		sess.Table("task_metric")
		if _, err := sess.Insert(&metrics); err != nil {
			return err
		}
	}
	return nil
}

func UpdateTask(t *model.TaskDTO) error {
	sess, err := newSession(true, "task")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = updateTask(sess, t); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func updateTask(sess *session, t *model.TaskDTO) error {
	return nil
}
