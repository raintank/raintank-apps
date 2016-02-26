package api

import (
	"github.com/Unknwon/macaron"
	"github.com/raintank/raintank-apps/server/model"
	"github.com/raintank/raintank-apps/server/sqlstore"
)

func GetAgents(ctx *macaron.Context, query model.GetAgentsQuery) {
	agents, err := sqlstore.GetAgents(&query)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, agents)
}

func GetAgentById(ctx *macaron.Context) {
	id := ctx.ParamsInt64(":id")
	owner := "admin"
	agent, err := sqlstore.GetAgentById(id, owner)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, agent)
}

func AddAgent(ctx *macaron.Context, agent model.AgentDTO) {
	//need to add suport for middelware context with AUTH/
	agent.Owner = "admin"
	err := sqlstore.UpdateAgent(&agent)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, agent)
}

func UpdateAgent(ctx *macaron.Context, agent model.AgentDTO) {
	//need to add suport for middelware context with AUTH/
	agent.Owner = "admin"
	err := sqlstore.UpdateAgent(&agent)
	if err != nil {
		log.Error(err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, agent)
}
