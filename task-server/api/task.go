package api

import (
	"fmt"

	"github.com/raintank/raintank-apps/task-server/api/rbody"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
)

func GetTaskById(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	owner := ctx.Owner
	task, err := sqlstore.GetTaskById(id, owner)
	if err != nil {
		log.Error(err)
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
	query.Owner = ctx.Owner
	tasks, err := sqlstore.GetTasks(&query)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("tasks", tasks))
}

func AddTask(ctx *Context, task model.TaskDTO) {
	task.Owner = ctx.Owner
	if task.Route.Type == model.RouteAny {
		// need to schedule the task to an agent.
		//TDOD: lookup least loded agent.
		task.Route.Config = map[string]interface{}{"id": int64(1)}
	}
	ok, err := task.Route.Validate()
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	if !ok {
		ctx.JSON(200, rbody.ErrResp(400, fmt.Errorf("invalid route config")))
		return
	}
	err = sqlstore.AddTask(&task)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ActiveSockets.EmitTask(&task, "taskAdd")
	taskCreate.Inc(1)
	ctx.JSON(200, rbody.OkResp("task", task))
}

func UpdateTask(ctx *Context, task model.TaskDTO) {
	task.Owner = ctx.Owner
	err := sqlstore.UpdateTask(&task)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("task", task))
}

func DeleteTask(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	owner := ctx.Owner
	existing, err := sqlstore.DeleteTask(id, owner)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	if existing != nil {
		ActiveSockets.EmitTask(existing, "taskRemove")
		taskDelete.Inc(1)
	}

	ctx.JSON(200, rbody.OkResp("task", nil))
}
