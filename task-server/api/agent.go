package api

import (
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
)

func GetAgents(ctx *Context, query model.GetAgentsQuery) {
	query.Owner = ctx.Owner
	agents, err := sqlstore.GetAgents(&query)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, agents)
}

func GetAgentById(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	owner := ctx.Owner
	agent, err := sqlstore.GetAgentById(id, owner)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	if agent == nil {
		ctx.JSON(404, "agent not found")
		return
	}
	ctx.JSON(200, agent)
}

func AddAgent(ctx *Context, agent model.AgentDTO) {
	if !agent.ValidName() {
		ctx.JSON(400, "invalde agent Name. must match /^[0-9a-Z_-]+$/")
		return
	}
	//need to add suport for middelware context with AUTH/
	agent.Owner = ctx.Owner
	err := sqlstore.UpdateAgent(&agent)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, agent)
}

func UpdateAgent(ctx *Context, agent model.AgentDTO) {
	if !agent.ValidName() {
		ctx.JSON(400, "invalde agent Name. must match /^[0-9a-Z_-]+$/")
		return
	}
	//need to add suport for middelware context with AUTH/
	agent.Owner = ctx.Owner
	err := sqlstore.UpdateAgent(&agent)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, agent)
}
