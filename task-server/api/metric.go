package api

import (
	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/raintank-apps/task-server/api/rbody"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
)

func GetMetrics(ctx *Context, query model.GetMetricsQuery) {
	query.OrgId = ctx.OrgId
	metrics, err := sqlstore.GetMetrics(&query)
	if err != nil {
		log.Error(3, err.Error())
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("metrics", metrics))
}
