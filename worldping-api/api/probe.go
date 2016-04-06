package api

import (
	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/raintank-apps/task-server/api/rbody"
	sModel "github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/worldping-api/model"
	"github.com/raintank/raintank-apps/worldping-api/task_client"
)

func GetProbes(ctx *Context, query model.GetProbesQuery) {
	query.OrgId = ctx.OrgId
	pQuery := sModel.GetAgentsQuery{
		Name:    query.Name,
		Metric:  "/worldping/*/*/ping/*",
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
		switch err.(type) {
		case rbody.ApiError:
			ctx.JSON(err.(rbody.ApiError).Code, err.(rbody.ApiError).Message)
		default:
			ctx.JSON(500, err)
		}
		return
	}
	ctx.JSON(200, agents)

}

func AddProbe(ctx *Context, p model.ProbeDTO) {
	p.OrgId = ctx.OrgId
	agent := &sModel.AgentDTO{
		OrgId:         ctx.OrgId,
		Name:          p.Name,
		Tags:          p.Tags,
		Public:        p.Public,
		Enabled:       p.Enabled,
		EnabledChange: p.EnabledChange,
		Online:        p.Online,
		OnlineChange:  p.OnlineChange,
	}
	err := task_client.Client.AddAgent(agent)
	if err != nil {
		log.Error(3, "api.AddProbe failed. %s", err)
		switch err.(type) {
		case rbody.ApiError:
			ctx.JSON(err.(rbody.ApiError).Code, err.(rbody.ApiError).Message)
		default:
			ctx.JSON(500, err)
		}
		return
	}
	p.Id = agent.Id
	ctx.JSON(200, p)
}

func UpdateProbe(ctx *Context, p model.ProbeDTO) {
	p.OrgId = ctx.OrgId
	agent := &sModel.AgentDTO{
		Id:            p.Id,
		OrgId:         ctx.OrgId,
		Name:          p.Name,
		Tags:          p.Tags,
		Public:        p.Public,
		Enabled:       p.Enabled,
		EnabledChange: p.EnabledChange,
		Online:        p.Online,
		OnlineChange:  p.OnlineChange,
	}
	err := task_client.Client.UpdateAgent(agent)
	if err != nil {
		log.Error(3, "api.UpdateProbe failed. %s", err)
		switch err.(type) {
		case rbody.ApiError:
			ctx.JSON(err.(rbody.ApiError).Code, err.(rbody.ApiError).Message)
		default:
			ctx.JSON(500, err)
		}
		return
	}

	ctx.JSON(200, p)

}

func GetProbeById(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	agent, err := task_client.Client.GetAgentById(id)
	if err != nil {
		log.Error(3, "api.GetProbeById failed. %s", err)
		switch err.(type) {
		case rbody.ApiError:
			ctx.JSON(err.(rbody.ApiError).Code, err.(rbody.ApiError).Message)
		default:
			ctx.JSON(500, err)
		}
		return
	}

	ctx.JSON(200, agent)
}

func DeleteProbe(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	err := task_client.Client.DeleteAgent(&sModel.AgentDTO{Id: id})
	if err != nil {
		log.Error(3, "api.DeleteProbe failed. %s", err)
		switch err.(type) {
		case rbody.ApiError:
			ctx.JSON(err.(rbody.ApiError).Code, err.(rbody.ApiError).Message)
		default:
			ctx.JSON(500, err)
		}
		return
	}
	ctx.JSON(200, "OK")
}
