package api

import (
	"fmt"
	"strings"

	"github.com/raintank/raintank-apps/task-server/api/rbody"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
	"github.com/raintank/worldping-api/pkg/log"
)

func GetTaskById(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	owner := ctx.OrgId
	task, err := sqlstore.GetTaskById(id, owner)
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	if task == nil {
		ctx.JSON(404, "task not found")
		return
	}
	ctx.JSON(200, rbody.OkResp("task", task))
}

func GetTasks(ctx *Context, query model.GetTasksQuery) {
	query.OrgId = ctx.OrgId

	// added for backwards compatibility.
	// if the user is looking for tasks for a specific metric type
	// translate that to a task with a specific type
	if query.Metric != "" {
		query.TaskType = strings.TrimRight(query.Metric, "/*")
	}
	tasks, err := sqlstore.GetTasks(&query)
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("tasks", tasks))
}

func AddTask(ctx *Context, task model.TaskDTO) {
	task.OrgId = ctx.OrgId

	ok, err := task.Route.Validate()
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	if !ok {
		ctx.JSON(200, rbody.ErrResp(400, fmt.Errorf("invalid route config")))
		return
	}

	err = sqlstore.AddTask(&task)
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	taskCreate.Inc(1)
	ctx.JSON(200, rbody.OkResp("task", task))
}

func UpdateTask(ctx *Context, task model.TaskDTO) {
	task.OrgId = ctx.OrgId

	ok, err := task.Route.Validate()
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	if !ok {
		ctx.JSON(200, rbody.ErrResp(400, fmt.Errorf("invalid route config")))
		return
	}

	err = sqlstore.UpdateTask(&task)
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("task", task))
}

func DeleteTask(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	owner := ctx.OrgId
	existing, err := sqlstore.DeleteTask(id, owner)
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	if existing != nil {
		ActiveSockets.EmitTask(existing, "taskRemove")
		taskDelete.Inc(1)
	}

	ctx.JSON(200, rbody.OkResp("task", nil))
}
