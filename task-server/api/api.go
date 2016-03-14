package api

import (
	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/binding"
	"github.com/op/go-logging"
	"github.com/raintank/met"

	"github.com/raintank/raintank-apps/task-server/model"
)

var log = logging.MustGetLogger("default")

var (
	taskCreate met.Count
	taskDelete met.Count
)

func Init(m *macaron.Macaron, adminKey string, metrics met.Backend) {
	m.Use(GetContextHandler())
	m.Use(Auth(adminKey))
	bind := binding.Bind

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
			m.Delete("/:id", DeleteTask)
		})
	})

	m.Get("/socket/:agent/:ver", socket)

	taskCreate = metrics.NewCount("api.tasks_create")
	taskDelete = metrics.NewCount("api.tasks_delete")
}

func index(ctx *macaron.Context) {
	ctx.JSON(200, "ok")
}
