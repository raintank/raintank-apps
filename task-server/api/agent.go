package api

import (
	"fmt"

	"github.com/raintank/raintank-apps/task-server/api/rbody"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
)

func GetAgents(ctx *Context, query model.GetAgentsQuery) {
	query.Owner = ctx.Owner
	agents, err := sqlstore.GetAgents(&query)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("agents", agents))
}

func GetAgentById(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	owner := ctx.Owner
	agent, err := sqlstore.GetAgentById(id, owner)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	if agent == nil {
		ctx.JSON(200, rbody.ErrResp(404, fmt.Errorf("agent not found")))
		return
	}
	ctx.JSON(200, rbody.OkResp("agent", agent))
}

func AddAgent(ctx *Context, agent model.AgentDTO) {
	if !agent.ValidName() {
		ctx.JSON(400, "invalde agent Name. must match /^[0-9a-Z_-]+$/")
		return
	}
	agent.Id = 0
	//need to add suport for middelware context with AUTH/
	agent.Owner = ctx.Owner
	err := sqlstore.AddAgent(&agent)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("agent", agent))
}

func UpdateAgent(ctx *Context, agent model.AgentDTO) {
	if !agent.ValidName() {
		ctx.JSON(200, rbody.ErrResp(400, fmt.Errorf("invalid agent Name. must match /^[0-9a-Z_-]+$/")))
		return
	}
	if agent.Id == 0 {
		ctx.JSON(200, rbody.ErrResp(400, fmt.Errorf("agent ID not set.")))
		return
	}
	//need to add suport for middelware context with AUTH/
	agent.Owner = ctx.Owner
	err := sqlstore.UpdateAgent(&agent)
	if err != nil {
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}
	ctx.JSON(200, rbody.OkResp("agent", agent))
}

func DeleteAgent(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	owner := ctx.Owner
	err := sqlstore.DeleteAgent(id, owner)
	if err != nil {
		if err == model.AgentNotFound {
			ctx.JSON(200, rbody.ErrResp(404, fmt.Errorf("agent not found")))
			return
		}
		log.Error(err)
		ctx.JSON(200, rbody.ErrResp(500, err))
		return
	}

	ActiveSockets.CloseSocketByAgentId(id)

	ctx.JSON(200, rbody.OkResp("agent", nil))
}
