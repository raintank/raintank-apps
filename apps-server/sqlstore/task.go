package sqlstore

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raintank/raintank-apps/apps-server/model"
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
	if query.Owner != 0 {
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
		sess.Join("INNER", []string{"task_metric", "tm"}, "task.id = tm.task_id").
			Where("tm.namespace=?", query.Metric)
		if query.MetricVersion == 0 {
			// get the latest version.
			sess.And("tm.version = (SELECT MAX(version) FROM metric WHERE namespace=? AND (owner=? or public=1) group by version)", query.Metric, query.Owner)
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

func GetTaskById(id int64, owner int64) (*model.TaskDTO, error) {
	sess, err := newSession(false, "task")
	if err != nil {
		return nil, err
	}
	return getTaskById(sess, id, owner)
}

func getTaskById(sess *session, id int64, owner int64) (*model.TaskDTO, error) {
	var t taskWithMetrics
	err := sess.Where("task.id=? AND owner=?", id, owner).Join("LEFT", "task_metric", "task.id = task_metric.task_id").Find(&t)
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
		//validate metrics
		mQuery := &model.GetMetricsQuery{
			Namespace: namespace,
			Owner:     t.Owner,
		}
		if ver != 0 {
			mQuery.Version = ver
		}
		matches, err := getMetrics(sess, mQuery)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return fmt.Errorf("no matching metric found.")
		}
		//Use the latest version available.
		if len(matches) > 1 {
			for _, m := range matches {
				if m.Version > ver {
					ver = m.Version
				}
			}
		}
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

	// add routeIndexes
	switch t.Route.Type {
	case model.RouteAny:
		idx := model.RouteByIdIndex{
			TaskId:  t.Id,
			AgentId: t.Route.Config["id"].(int64),
			Created: time.Now(),
		}
		if _, err := sess.Insert(&idx); err != nil {
			return err
		}
	case model.RouteByTags:
		tagRoutes := make([]*model.RouteByTagIndex, len(t.Route.Config["tags"].([]string)))
		for i, tag := range t.Route.Config["tags"].([]string) {
			tagRoutes[i] = &model.RouteByTagIndex{
				TaskId:  t.Id,
				Tag:     tag,
				Created: time.Now(),
			}
		}
		if _, err := sess.Insert(&tagRoutes); err != nil {
			return err
		}
	case model.RouteByIds:
		idxs := make([]*model.RouteByIdIndex, len(t.Route.Config["ids"].([]int64)))
		for i, id := range t.Route.Config["ids"].([]int64) {
			idxs[i] = &model.RouteByIdIndex{
				TaskId:  t.Id,
				AgentId: id,
				Created: time.Now(),
			}
		}
		if _, err := sess.Insert(&idxs); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown routeType")
	}

	return nil
}

func UpdateTask(t *model.TaskDTO) error {
	sess, err := newSession(true, "task")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	err = updateTask(sess, t)
	if err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func updateTask(sess *session, t *model.TaskDTO) error {
	return nil
}

func GetAgentTasks(agent *model.AgentDTO) ([]*model.TaskDTO, error) {
	sess, err := newSession(true, "task")
	if err != nil {
		return nil, err
	}
	defer sess.Cleanup()
	tasks, err := getAgentTasks(sess, agent)
	if err != nil {
		return nil, err
	}
	sess.Complete()
	return tasks, nil
}

func getAgentTasks(sess *session, agent *model.AgentDTO) ([]*model.TaskDTO, error) {
	var tasks taskWithMetrics

	// Get taskIds (we could do this with an INNER join on a subquery, but xorm makes that hard to do.)
	type taskIdRow struct {
		TaskId int64
	}
	taskIds := make([]*taskIdRow, 0)
	rawQuery := "SELECT task_id FROM route_by_id_index where agent_id = ?"
	rawParams := make([]interface{}, 0)
	rawParams = append(rawParams, agent.Id)
	if len(agent.Tags) > 0 {
		p := make([]string, len(agent.Tags))
		for i, t := range agent.Tags {
			p[i] = "?"
			rawParams = append(rawParams, t)
		}
		rawQuery = fmt.Sprintf("%s UNION SELECT task_id FROM route_by_tag_index where tag IN (%s)", rawQuery, strings.Join(p, ","))
	}
	err := sess.Sql(rawQuery, rawParams...).Find(&taskIds)
	if err != nil {
		return nil, err
	}

	if len(taskIds) == 0 {
		return nil, nil
	}
	tid := make([]int64, len(taskIds))
	for i, t := range taskIds {
		tid[i] = t.TaskId
	}
	sess.Table("task")
	sess.Join("LEFT", "task_metric", "task.id = task_metric.task_id")
	sess.In("task.id", tid)

	err = sess.Find(&tasks)
	return tasks.ToTaskDTO(), err
}

func DeleteTask(id int64, owner int64) (*model.TaskDTO, error) {
	sess, err := newSession(true, "task")
	if err != nil {
		return nil, err
	}
	defer sess.Cleanup()
	existing, err := deleteTask(sess, id, owner)
	if err != nil {
		return nil, err
	}
	sess.Complete()
	return existing, nil
}

func deleteTask(sess *session, id int64, owner int64) (*model.TaskDTO, error) {
	existing, err := getTaskById(sess, id, owner)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	deletes := []string{
		"DELETE FROM task WHERE id = ?",
		"DELETE FROM task_metric WHERE task_id = ?",
		"DELETE from route_by_id_index where task_id = ?",
		"DELETE from route_by_tag_index where task_id = ?",
	}

	for _, sql := range deletes {
		_, err := sess.Exec(sql, id)
		if err != nil {
			return nil, err
		}
	}
	return existing, nil
}
