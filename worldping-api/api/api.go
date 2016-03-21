package api

import (
	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/binding"
	"github.com/raintank/met"
	"github.com/raintank/raintank-apps/worldping-api/model"
)

func Init(m *macaron.Macaron, adminKey string, metrics met.Backend) {
	m.Use(GetContextHandler())
	bind := binding.Bind

	InitEndpointMetrics(metrics)
	//InitProbeMetrics(metrics)

	m.Get("/", index)
	m.Group("/api", func() {
		m.Group("/endpoints", func() {
			m.Combo("/").
				Get(bind(model.GetEndpointsQuery{}), GetEndpoints).
				Post(bind(model.EndpointDTO{}), AddEndpoint).
				Put(bind(model.EndpointDTO{}), UpdateEndpoint)
			m.Get("/:id", GetEndpointById)
			m.Delete("/:id", DeleteEndpoint)
		})

		m.Group("/probes", func() {
			m.Combo("/").
				Get(bind(model.GetProbesQuery{}), GetProbes).
				Post(bind(model.ProbeDTO{}), AddProbe).
				Put(bind(model.ProbeDTO{}), UpdateProbe)
			m.Get("/:id", GetProbeById)
			m.Delete("/:id", DeleteProbe)
		})

		/*
			m.Group("/admin", func() {
				m.Get("/", index)

				m.Get("/quota/:owner", GetQuotas)
				m.Put("/quota/:owner/endpoint", bind(model.UpdateQuotaCmd{}), UpdateEndpointQuota)

			}, RequireAdmin())
		*/
	}, Auth(adminKey))
}

func index(ctx *macaron.Context) {
	ctx.JSON(200, "ok")
}
