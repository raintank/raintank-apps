package api

import (
	"github.com/Unknwon/macaron"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("default")

func InitRoutes(m *macaron.Macaron, adminKey string) {
	m.Use(GetContextHandler())
	m.Use(Auth(adminKey))

	m.Get("/", index)
	m.Post("/metrics", Metrics)
	m.Any("/graphite/*", GraphiteProxy)
	//m.Any("/elasticsearch/*", ElasticsearchProxy)
}

func index(ctx *macaron.Context) {
	ctx.JSON(200, "ok")
}
