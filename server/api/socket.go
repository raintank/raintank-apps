package api

import (
	"errors"

	"github.com/Unknwon/macaron"
	"github.com/gorilla/websocket"

	"github.com/raintank/raintank-apps/server/agent_session"
	"github.com/raintank/raintank-apps/server/model"
	"github.com/raintank/raintank-apps/server/sqlstore"
)

var upgrader = websocket.Upgrader{} // use default options

func connectedAgent(agentName string, owner string) (*model.AgentDTO, error) {
	if agentName == "" {
		return nil, errors.New("agent name not specified.")
	}
	q := model.GetAgentsQuery{
		Name:  agentName,
		Owner: owner,
	}
	agents, err := sqlstore.GetAgents(&q)
	if err != nil {
		return nil, err
	}
	if len(agents) < 1 {
		return nil, errors.New("agent not found.")
	}
	return agents[0], nil
}

func socket(ctx *macaron.Context) {
	agentName := ctx.Params(":agent")
	agentVer := ctx.ParamsInt64(":ver")
	//TODO: add auth
	owner := "admin"
	agent, err := connectedAgent(agentName, owner)
	if err != nil {
		log.Debugf("agent cant connect. %s", err)
		ctx.JSON(400, err.Error())
		return
	}

	c, err := upgrader.Upgrade(ctx.Resp, ctx.Req.Request, nil)
	if err != nil {
		log.Errorf("upgrade:", err)
		return
	}

	log.Debugf("agent %s connected.", agent.Name)

	sess := agent_session.NewSession(agent, agentVer, c)
	sess.Start()
	//block until connection closes.
	<-sess.Done
}
