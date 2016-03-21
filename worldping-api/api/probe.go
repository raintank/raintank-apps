package api

import (
	"github.com/grafana/grafana/pkg/log"
	sModel "github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/worldping-api/model"
	"github.com/raintank/raintank-apps/worldping-api/task_client"
)

func GetProbes(ctx *Context, query model.GetProbesQuery) {
	query.Owner = ctx.Owner
	pQuery := sModel.GetAgentsQuery{
		Name:    query.Name,
		Enabled: query.Enabled,
		Public:  query.Public,
		Tag:     query.Tag,
		OrderBy: query.OrderBy,
		Limit:   query.Limit,
		Page:    query.Page,
	}

	agents, err := task_client.Client.GetAgents(&pQuery)
	if err != nil {
		log.Error(3, "api.GetProbes failed. %s", err)
		ctx.JSON(500, err)
	}
	ctx.JSON(200, agents)

}

func AddProbe(ctx *Context, p model.ProbeDTO) {
	p.Owner = ctx.Owner
	ctx.JSON(200, "OK")
}

func UpdateProbe(ctx *Context, p model.ProbeDTO) {
	p.Owner = ctx.Owner
	ctx.JSON(200, "OK")

}

func GetProbeById(ctx *Context) {
	//id := ctx.ParamsInt64(":id")
	ctx.JSON(200, "OK")
}

func DeleteProbe(ctx *Context) {
	//id := ctx.ParamsInt64(":id")
	ctx.JSON(200, "OK")

}
