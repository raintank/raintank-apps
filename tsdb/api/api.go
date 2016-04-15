package api

import (
	"github.com/Unknwon/macaron"
)

func InitRoutes(m *macaron.Macaron, adminKey string) {
	m.Use(GetContextHandler())
	m.Use(Auth(adminKey))

	m.Get("/", index)
	m.Post("/metrics", Metrics)
	m.Post("/events", Events)
	m.Any("/graphite/*", GraphiteProxy)
	m.Any("/elasticsearch/*", ElasticsearchProxy)
}

func index(ctx *macaron.Context) {
	ctx.JSON(200, "ok")
}
