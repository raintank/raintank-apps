package api

import (
	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/binding"
	"github.com/op/go-logging"

	"github.com/raintank/raintank-apps/server/model"
)

var log = logging.MustGetLogger("default")

func InitRoutes(m *macaron.Macaron) {
	bind := binding.Bind

	m.Get("/socket/:agent/:ver", socket)
	m.Get("/", index)
	m.Group("/api", func() {
		m.Group("/agents", func() {
			m.Combo("/").
				Get(bind(model.GetAgentsQuery{}), GetAgents).
				Post(bind(model.AgentDTO{}), AddAgent).
				Put(bind(model.AgentDTO{}), UpdateAgent)
			m.Get("/:id", GetAgentById)
		})

		m.Get("/metrics", bind(model.GetMetricsQuery{}), GetMetrics)

		m.Group("/tasks", func() {
			m.Combo("/").
				Get(bind(model.GetTasksQuery{}), GetTasks).
				Post(bind(model.TaskDTO{}), AddTask).
				Put(bind(model.TaskDTO{}), UpdateTask)
			m.Get("/:id", GetTaskById)
		})
	})
}

func index(ctx *macaron.Context) {
	ctx.JSON(200, "ok")
}
