package api

import (
	"github.com/Unknwon/macaron"
	"github.com/raintank/raintank-apps/server/model"
	"github.com/raintank/raintank-apps/server/sqlstore"
)

func GetMetrics(ctx *macaron.Context, query model.GetMetricsQuery) {
	query.Owner = "admin"
	metrics, err := sqlstore.GetMetrics(&query)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, metrics)
}
