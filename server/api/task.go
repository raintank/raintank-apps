package api

import (
	"github.com/Unknwon/macaron"
	"github.com/raintank/raintank-apps/server/model"
	"github.com/raintank/raintank-apps/server/sqlstore"
)

func GetTaskById(ctx *macaron.Context) {
	id := ctx.ParamsInt64(":id")
	owner := "admin"
	task, err := sqlstore.GetTaskById(id, owner)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	if task == nil {
		ctx.JSON(404, "task not found")
		return
	}
	ctx.JSON(200, task)
}

func GetTasks(ctx *macaron.Context, query model.GetTasksQuery) {
	query.Owner = "admin"
	tasks, err := sqlstore.GetTasks(&query)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, tasks)
}

func AddTask(ctx *macaron.Context, task model.TaskDTO) {
	task.Owner = "admin"
	if task.Route.Type == model.RouteAny {
		// need to schedule the task to an agent.
		//TDOD: lookup least loded agent.
		task.Route.Config = map[string]interface{}{"id": int64(1)}
	}
	ok, err := task.Route.Validate()
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	if !ok {
		ctx.JSON(400, "invalid route config")
		return
	}
	err = sqlstore.AddTask(&task)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ActiveSockets.EmitTask(&task, "taskAdd")
	ctx.JSON(200, task)
}

func UpdateTask(ctx *macaron.Context, task model.TaskDTO) {
	task.Owner = "admin"
	err := sqlstore.UpdateTask(&task)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, task)
}

func DeleteTask(ctx *macaron.Context) {
	id := ctx.ParamsInt64(":id")
	owner := "admin"
	existing, err := sqlstore.DeleteTask(id, owner)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	if existing != nil {
		ActiveSockets.EmitTask(existing, "taskRemove")
	}

	ctx.JSON(200, "ok")
}
