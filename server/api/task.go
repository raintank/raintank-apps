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
	tasks, err := sqlstore.GetTasks(query)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, tasks)
}

func AddTask(ctx *macaron.Context, task model.Task) {
	task.Owner = "admin"
	err := sqlstore.AddTask(&task)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, task)
}

func UpdateTask(ctx *macaron.Context, task model.Task) {
	task.Owner = "admin"
	err := sqlstore.UpdateTask(&task)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, task)
}
