package sqlstore

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raintank/raintank-apps/task-server/model"
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
				OrgId:    r.OrgId,
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
	if query.OrgId != 0 {
		sess.Where("task.org_id = ?", query.OrgId)
	}
	if query.Enabled != "" {
		enabled, err := strconv.ParseBool(query.Enabled)
		if err != nil {
			return nil, err
		}
		sess.Where("task.enabled=?", enabled)
	}

	if query.Name != "" {
		sess.And("task.name like ?", query.Name)
	}

	if query.Metric != "" {
		sess.Join("INNER", []string{"task_metric", "tm"}, "task.id = tm.task_id").
			Where("tm.namespace=?", query.Metric)
	}
	if query.OrderBy == "" {
		query.OrderBy = "name"
	}
	if query.Limit == 0 {
		query.Limit = 50
	}
	if query.Page == 0 {
		query.Page = 1
	}
	sess.Asc(query.OrderBy).Limit(query.Limit, (query.Page-1)*query.Limit)

	sess.Cols(
		"task.id",
		"task.name",
		"task.org_id",
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

func GetTaskById(id int64, orgId int64) (*model.TaskDTO, error) {
	sess, err := newSession(false, "task")
	if err != nil {
		return nil, err
	}
	return getTaskById(sess, id, orgId)
}

func getTaskById(sess *session, id int64, orgId int64) (*model.TaskDTO, error) {
	var t taskWithMetrics
	err := sess.Where("task.id=? AND org_id=?", id, orgId).Join("LEFT", "task_metric", "task.id = task_metric.task_id").Find(&t)
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
		OrgId:    t.OrgId,
		Interval: t.Interval,
		Enabled:  t.Enabled,
		Config:   t.Config,
		Route:    t.Route,
		Created:  time.Now(),
		Updated:  time.Now(),
	}
	sess.UseBool("enabled")
	if _, err := sess.Insert(&task); err != nil {
		return err
	}
	t.Created = task.Created
	t.Updated = task.Updated
	t.Id = task.Id

	// handle metrics.
	metrics := make([]*model.TaskMetric, 0, len(t.Metrics))
	for namespace, ver := range t.Metrics {
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
	return addTaskRoute(sess, t)

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
	existing, err := getTaskById(sess, t.Id, t.OrgId)
	if err != nil {
		return err
	}
	if existing == nil {
		return model.TaskNotFound
	}
	task := model.Task{
		Id:       t.Id,
		Name:     t.Name,
		OrgId:    t.OrgId,
		Interval: t.Interval,
		Enabled:  t.Enabled,
		Config:   t.Config,
		Route:    t.Route,
		Created:  existing.Created,
		Updated:  time.Now(),
	}
	sess.UseBool("enabled")
	_, err = sess.Id(task.Id).Update(&task)
	if err != nil {
		return err
	}
	t.Updated = task.Updated

	// Update taskMetrics
	metricsToAdd := make([]*model.TaskMetric, 0)
	metricsToDel := make([]*model.TaskMetric, 0)
	metricsMap := make(map[string]*model.TaskMetric)
	seenMetrics := make(map[string]struct{})

	for m, v := range existing.Metrics {
		metricsMap[fmt.Sprintf("%s:%d", m, v)] = &model.TaskMetric{
			TaskId:    t.Id,
			Namespace: m,
			Version:   v,
		}
	}
	for m, v := range t.Metrics {
		key := fmt.Sprintf("%s:%d", m, v)
		seenMetrics[key] = struct{}{}
		if _, ok := metricsMap[key]; !ok {
			metricsToAdd = append(metricsToAdd, &model.TaskMetric{
				TaskId:    t.Id,
				Namespace: m,
				Version:   v,
				Created:   time.Now(),
			})
		}
	}

	for key, m := range metricsMap {
		if _, ok := seenMetrics[key]; !ok {
			metricsToDel = append(metricsToDel, m)
		}
	}

	if len(metricsToDel) > 0 {
		_, err := sess.Delete(&metricsToDel)
		if err != nil {
			return err
		}
	}
	if len(metricsToAdd) > 0 {
		_, err := sess.Insert(&metricsToAdd)
		if err != nil {
			return err
		}
	}

	// handle task routes.
	if existing.Route.Type != t.Route.Type {
		if err := deleteTaskRoute(sess, existing); err != nil {
			return err
		}
		if err := addTaskRoute(sess, t); err != nil {
			return err
		}
	} else {
		switch t.Route.Type {
		case model.RouteAny:
			//no need to do anything.
		case model.RouteByTags:
			existingTags := make(map[string]struct{})
			tagsToAdd := make([]string, 0)
			tagsToDel := make([]string, 0)
			currentTags := make(map[string]struct{})

			for _, tag := range existing.Route.Config["tags"].([]string) {
				existingTags[tag] = struct{}{}
			}
			for _, tag := range t.Route.Config["tags"].([]string) {
				currentTags[tag] = struct{}{}
				if _, ok := existingTags[tag]; !ok {
					tagsToAdd = append(tagsToAdd, tag)
				}
			}
			for tag := range existingTags {
				if _, ok := currentTags[tag]; !ok {
					tagsToDel = append(tagsToDel, tag)
				}
			}
			if len(tagsToDel) > 0 {
				tagRoutes := make([]*model.RouteByTagIndex, len(tagsToDel))
				for i, tag := range tagsToDel {
					tagRoutes[i] = &model.RouteByTagIndex{
						TaskId: t.Id,
						Tag:    tag,
					}
				}
				_, err := sess.Delete(&tagRoutes)
				if err != nil {
					return err
				}
			}
			if len(tagsToAdd) > 0 {
				tagRoutes := make([]*model.RouteByTagIndex, len(tagsToAdd))
				for i, tag := range tagsToAdd {
					tagRoutes[i] = &model.RouteByTagIndex{
						TaskId:  t.Id,
						Tag:     tag,
						Created: time.Now(),
					}
				}
				_, err := sess.Insert(&tagRoutes)
				if err != nil {
					return err
				}
			}

		case model.RouteByIds:
			existingIds := make(map[int64]struct{})
			idsToAdd := make([]int64, 0)
			idsToDel := make([]int64, 0)
			currentIds := make(map[int64]struct{})

			for _, id := range existing.Route.Config["ids"].([]int64) {
				existingIds[id] = struct{}{}
			}
			for _, id := range t.Route.Config["ids"].([]int64) {
				currentIds[id] = struct{}{}
				if _, ok := existingIds[id]; !ok {
					idsToAdd = append(idsToAdd, id)
				}
			}
			for id := range existingIds {
				if _, ok := currentIds[id]; !ok {
					idsToDel = append(idsToDel, id)
				}
			}
			if len(idsToDel) > 0 {
				idRoutes := make([]*model.RouteByIdIndex, len(idsToDel))
				for i, id := range idsToDel {
					idRoutes[i] = &model.RouteByIdIndex{
						TaskId:  t.Id,
						AgentId: id,
					}
				}
				_, err := sess.Delete(&idRoutes)
				if err != nil {
					return err
				}
			}
			if len(idsToAdd) > 0 {
				idRoutes := make([]*model.RouteByIdIndex, len(idsToAdd))
				for i, id := range idsToAdd {
					idRoutes[i] = &model.RouteByIdIndex{
						TaskId:  t.Id,
						AgentId: id,
						Created: time.Now(),
					}
				}
				_, err := sess.Insert(&idRoutes)
				if err != nil {
					return err
				}
			}
		default:
			return model.UnknownRouteType
		}
	}

	return nil
}

func addTaskRoute(sess *session, t *model.TaskDTO) error {
	switch t.Route.Type {
	case model.RouteAny:
		idx := model.RouteByAnyIndex{
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
		return model.UnknownRouteType
	}
	return nil
}

func deleteTaskRoute(sess *session, t *model.TaskDTO) error {
	deletes := []string{
		"DELETE from route_by_id_index where task_id = ?",
		"DELETE from route_by_tag_index where task_id = ?",
		"DELETE from route_by_any_index where task_id = ?",
	}
	for _, sql := range deletes {
		_, err := sess.Exec(sql, t.Id)
		if err != nil {
			return err
		}
	}
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
	rawQuery := "SELECT task_id FROM route_by_id_index where agent_id = ? UNION SELECT task_id from route_by_any_index where agent_id = ?"
	rawParams := make([]interface{}, 0)
	rawParams = append(rawParams, agent.Id, agent.Id)
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

func DeleteTask(id int64, orgId int64) (*model.TaskDTO, error) {
	sess, err := newSession(true, "task")
	if err != nil {
		return nil, err
	}
	defer sess.Cleanup()
	existing, err := deleteTask(sess, id, orgId)
	if err != nil {
		return nil, err
	}
	sess.Complete()
	return existing, nil
}

func deleteTask(sess *session, id int64, orgId int64) (*model.TaskDTO, error) {
	existing, err := getTaskById(sess, id, orgId)
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
		"DELETE from route_by_any_index where task_id = ?",
	}

	for _, sql := range deletes {
		_, err := sess.Exec(sql, id)
		if err != nil {
			return nil, err
		}
	}
	return existing, nil
}
