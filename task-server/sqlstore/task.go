package sqlstore

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/raintank/raintank-apps/task-server/event"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/worldping-api/pkg/log"
)

func GetTasks(query *model.GetTasksQuery) ([]*model.TaskDTO, error) {
	sess, err := newSession(false, "task")
	if err != nil {
		return nil, err
	}
	return getTasks(sess, query)
}

func getTasks(sess *session, query *model.GetTasksQuery) ([]*model.TaskDTO, error) {
	var t []*model.TaskDTO
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
		"task.task_type",
		"task.org_id",
		"task.enabled",
		"task.interval",
		"task.config",
		"task.route",
		"task.created",
		"task.updated",
	)
	err := sess.Find(&t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func GetTaskById(id int64, orgId int64) (*model.TaskDTO, error) {
	sess, err := newSession(false, "task")
	if err != nil {
		return nil, err
	}
	return getTaskById(sess, id, orgId)
}

func getTaskById(sess *session, id int64, orgId int64) (*model.TaskDTO, error) {
	var t *model.TaskDTO
	sess.Where("task.id=? AND org_id=?", id, orgId)

	err := sess.Find(&t)
	if err != nil {
		return nil, err
	}
	return t, nil
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
	event.Publish(&event.TaskCreated{Ts: time.Now(), Payload: t}, 0)
	return nil
}

func addTask(sess *session, t *model.TaskDTO) error {
	task := model.Task{
		Name:     t.Name,
		TaskType: t.TaskType,
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

	// add routeIndexes
	return addTaskRoute(sess, t)

}

func taskRouteAnyCandidates(sess *session, tid int64) ([]int64, error) {
	// get Candidate Agents.
	candidates := make([]struct{ AgentId int64 }, 0)
	err := sess.Sql(`SELECT id as agent_id from agent where agent.online=1`).Find(&candidates)
	if err != nil {
		return nil, err
	}

	resp := make([]int64, len(candidates))
	for i, c := range candidates {
		resp[i] = c.AgentId
	}
	return resp, nil
}

func UpdateTask(t *model.TaskDTO) error {
	sess, err := newSession(true, "task")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	events, err := updateTask(sess, t)
	if err != nil {
		return err
	}
	sess.Complete()
	for _, e := range events {
		event.Publish(e, 0)
	}
	return nil
}

func updateTask(sess *session, t *model.TaskDTO) ([]event.Event, error) {
	events := make([]event.Event, 0)
	existing, err := getTaskById(sess, t.Id, t.OrgId)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, model.TaskNotFound
	}
	task := model.Task{
		Id:       t.Id,
		Name:     t.Name,
		TaskType: t.TaskType,
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
		return nil, err
	}
	t.Updated = task.Updated

	// handle task routes.
	if existing.Route.Type != t.Route.Type {
		if err := deleteTaskRoute(sess, existing); err != nil {
			return nil, err
		}
		if err := addTaskRoute(sess, t); err != nil {
			return nil, err
		}
	} else {
		switch t.Route.Type {
		case model.RouteAny:
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
					return nil, err
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
					return nil, err
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
					return nil, err
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
					return nil, err
				}
			}
		default:
			return nil, model.UnknownRouteType
		}
	}
	e := new(event.TaskUpdated)
	e.Ts = time.Now()
	e.Payload.Last = t
	e.Payload.Current = t
	events = append(events, e)
	return events, nil
}

func addTaskRoute(sess *session, t *model.TaskDTO) error {
	switch t.Route.Type {
	case model.RouteAny:
		candidates, err := taskRouteAnyCandidates(sess, t.Id)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return fmt.Errorf("No agent found that can provide all requested metrics.")
		}

		idx := model.RouteByAnyIndex{
			TaskId:  t.Id,
			AgentId: candidates[rand.Intn(len(candidates))],
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
				OrgId:   t.OrgId,
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

func RelocateRouteAnyTasks(agent *model.AgentDTO) error {
	sess, err := newSession(true, "task")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	events, err := relocateRouteAnyTasks(sess, agent)
	if err != nil {
		return err
	}
	sess.Complete()
	for _, e := range events {
		event.Publish(e, 0)
	}
	return nil
}

func relocateRouteAnyTasks(sess *session, agent *model.AgentDTO) ([]event.Event, error) {
	events := make([]event.Event, 0)
	// get list of tasks.
	var tasks []*model.TaskDTO
	sess.Join("INNER", "route_by_any_index", "route_by_any_index.task_id = task.id").Where("route_by_any_index.agent_id=?", agent.Id)
	err := sess.Find(&tasks)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	for _, t := range tasks {
		candidates, err := taskRouteAnyCandidates(sess, t.Id)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			log.Error(3, "Cant re-locate task %d, no online agents capable of providing requested metrics.", t.Id)
			continue
		}
		newAgent := candidates[rand.Intn(len(candidates))]
		if newAgent == agent.Id {
			log.Debug("No need to re-allocated task as the agent it was running on is back online")
			continue
		}
		_, err = sess.Exec("UPDATE route_by_any_index set agent_id=? where task_id=?", newAgent, t.Id)
		if err != nil {
			return nil, err
		}
		log.Info("Task %d rescheduled to agent %d", t.Id, newAgent)
		e := new(event.TaskUpdated)
		e.Ts = time.Now()
		e.Payload.Last = t
		e.Payload.Current = t
		events = append(events, e)
	}
	return events, nil
}

func GetAgentTasks(agent *model.AgentDTO) ([]*model.TaskDTO, error) {
	sess, err := newSession(false, "task")
	if err != nil {
		return nil, err
	}
	return getAgentTasks(sess, agent)
}

func getAgentTasks(sess *session, agent *model.AgentDTO) ([]*model.TaskDTO, error) {
	var tasks []*model.TaskDTO

	type taskIdRow struct {
		TaskId int64
	}
	taskIds := make([]*taskIdRow, 0)
	rawQuery := "SELECT task_id FROM route_by_id_index where agent_id = ? UNION SELECT task_id from route_by_any_index where agent_id = ?"
	rawParams := make([]interface{}, 0)
	rawParams = append(rawParams, agent.Id, agent.Id)

	q := `SELECT
	           DISTINCT(idx.task_id)
	        FROM route_by_tag_index AS idx
	        INNER JOIN agent_tag on idx.org_id=agent_tag.org_id and idx.tag = agent_tag.tag
	        WHERE agent_tag.agent_id = ?`
	rawParams = append(rawParams, agent.Id)
	rawQuery = fmt.Sprintf("%s UNION %s", rawQuery, q)
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
	sess.Where("task.enabled=1")
	sess.In("task.id", tid)
	sess.Cols("id", "name", "config", "interval", "org_id", "enabled", "route", "created", "updated", "task_type")

	err = sess.Find(&tasks)
	if err != nil {
		return nil, err
	}
	return tasks, err
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

	event.Publish(&event.TaskDeleted{Ts: time.Now(), Payload: existing}, 0)

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
