package api

import (
	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/binding"
	"github.com/raintank/met"
	"github.com/raintank/raintank-apps/task-server/api/rbody"
	"github.com/raintank/raintank-apps/task-server/model"
)

var (
	taskCreate      met.Count
	taskDelete      met.Count
	agentsConnected met.Gauge
)

func NewApi(adminKey string, metrics met.Backend) *macaron.Macaron {
	m := macaron.Classic()
	m.Use(macaron.Renderer())
	m.Use(GetContextHandler())
	bind := binding.Bind

	m.Get("/", heartbeat)
	m.Group("/api/v1", func() {
		m.Get("/", heartbeat)
		m.Group("/agents", func() {
			m.Combo("/").
				Get(bind(model.GetAgentsQuery{}), GetAgents).
				Post(AgentQuota(), bind(model.AgentDTO{}), AddAgent).
				Put(bind(model.AgentDTO{}), UpdateAgent)
			m.Get("/:id", GetAgentById)
			m.Delete("/:id", DeleteAgent)
		})

		m.Group("/tasks", func() {
			m.Combo("/").
				Get(bind(model.GetTasksQuery{}), GetTasks).
				Post(bind(model.TaskDTO{}), TaskQuota(), AddTask).
				Put(bind(model.TaskDTO{}), UpdateTask)
			m.Get("/:id", GetTaskById)
			m.Delete("/:id", DeleteTask)
		})
		m.Get("/socket/:agent/:ver", socket)
	}, Auth(adminKey))

	taskCreate = metrics.NewCount("api.tasks_create")
	taskDelete = metrics.NewCount("api.tasks_delete")
	agentsConnected = metrics.NewGauge("api.agents_connected", 0)
	return m
}

func heartbeat(ctx *macaron.Context) {
	ctx.JSON(200, rbody.OkResp("heartbeat", nil))
}
