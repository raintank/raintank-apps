package api

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/grafana/metrictank/stats"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/task-server/agent_session"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
	"github.com/raintank/worldping-api/pkg/log"
)

var (
	taskServerAgentConnectionsActiveCount   = stats.NewGauge64("agent.connections.active")
	taskServerAgentConnectionsFailedCount   = stats.NewCounter64("agent.connections.failed")
	taskServerAgentConnectionsAcceptedCount = stats.NewCounter64("agent.connections.accepted")
	taskServerAgentAutoCreateSuccessCount   = stats.NewCounter64("agent.autocreate.success")
	taskServerAgentAutoCreateFailedCount    = stats.NewCounter64("agent.autocreate.failed")
)

var upgrader = websocket.Upgrader{} // use default options

type socketList struct {
	sync.RWMutex
	Sockets map[int64]*agent_session.AgentSession
}

func (s *socketList) CloseAll() {
	s.Lock()
	for _, sock := range s.Sockets {
		sock.Close()
	}
	s.Sockets = make(map[int64]*agent_session.AgentSession, 0)
	s.Unlock()
}

func (s *socketList) EmitTask(task *model.TaskDTO, event string) error {
	log.Debug("sending %s task event to connected agents.", event)
	agents, err := sqlstore.GetAgentsForTask(task)
	log.Debug("Task has %d agents. %v", len(agents), agents)
	if err != nil {
		return err
	}
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}
	e := &message.Event{
		Event:   event,
		Payload: body,
	}
	sent := false
	s.Lock()

	for _, id := range agents {
		if as, ok := s.Sockets[id]; ok {
			log.Debug("sending %s event to agent %d", event, id)
			as.SocketSession.Emit(e)
			sent = true
		} else {
			log.Debug("agent %d is not connected to this server.", id)
		}
	}
	s.Unlock()
	if !sent {
		log.Debug("no connected agents for task %d.", task.Id)
	}
	return nil
}

func (s *socketList) NewSocket(a *agent_session.AgentSession) {
	s.Lock()
	existing, ok := s.Sockets[a.Agent.Id]
	if ok {
		log.Debug("new connection for agent %d - %s, closing existing session", a.Agent.Id, a.Agent.Name)
		existing.Close()
	}
	log.Debug("Agent %d is connected to this server.", a.Agent.Id)
	s.Sockets[a.Agent.Id] = a
	taskServerAgentConnectionsAcceptedCount.Inc()
	taskServerAgentConnectionsActiveCount.Inc()
	s.Unlock()
}

func (s *socketList) DeleteSocket(a *agent_session.AgentSession) {
	s.Lock()
	s.deleteSocket(a.Agent.Id)
	s.Unlock()
}

func (s *socketList) deleteSocket(id int64) {
	delete(s.Sockets, id)
	taskServerAgentConnectionsActiveCount.Dec()
}

func (s *socketList) CloseSocket(a *agent_session.AgentSession) {
	s.Lock()
	existing, ok := s.Sockets[a.Agent.Id]
	if ok {
		existing.Close()
		log.Debug("removing session for Agent %d from socketList.", a.Agent.Id)
		s.deleteSocket(a.Agent.Id)
	}
	s.Unlock()
}

func (s *socketList) CloseSocketByAgentId(id int64) {
	s.Lock()
	existing, ok := s.Sockets[id]
	if ok {
		existing.Close()
		log.Debug("CloseSocketByAgentId: removing session for Agent %d from socketList.", id)
		s.deleteSocket(id)
	}
	s.Unlock()
}

func newSocketList() *socketList {
	return &socketList{
		Sockets: make(map[int64]*agent_session.AgentSession),
	}
}

var ActiveSockets *socketList

func init() {
	ActiveSockets = newSocketList()
}

func connectedAgent(agentName string, orgID int64) (*model.AgentDTO, error) {
	if agentName == "" {
		return nil, errors.New("connectedAgent: agent name not specified")
	}
	q := model.GetAgentsQuery{
		Name:  agentName,
		OrgId: orgID,
	}
	agents, err := sqlstore.GetAgents(&q)
	if err != nil {
		return nil, err
	}
	if len(agents) < 1 {
		log.Debug("connectedAgent: Agent %s not found, attempting to create.", agentName)

		// auto add agent
		anAgent := model.AgentDTO{
			Name:    agentName,
			OrgId:   orgID,
			Enabled: true,
			Public:  true,
		}
		addAgentErr := sqlstore.AddAgent(&anAgent)
		if addAgentErr != nil {
			log.Error(3, addAgentErr.Error())
			taskServerAgentConnectionsFailedCount.Inc()
			taskServerAgentAutoCreateFailedCount.Inc()
			return nil, errors.New("connectedAgent: failed to auto create agent.")
		}
		log.Debug("connectedAgent: Agent %s created.", agentName)

		agentsAfter, err := sqlstore.GetAgents(&q)
		if err != nil {
			return nil, err
		}
		if len(agentsAfter) < 1 {
			taskServerAgentConnectionsFailedCount.Inc()
			taskServerAgentAutoCreateFailedCount.Inc()
			return nil, errors.New("connectedAgent: agent not found after autocreate.")
		}
		log.Debug("connectedAgent: Allowing new Agent %s.", agentName)
		taskServerAgentConnectionsAcceptedCount.Inc()
		taskServerAgentAutoCreateSuccessCount.Inc()

		return agentsAfter[0], nil
	}

	return agents[0], nil
}

func socket(ctx *Context) {
	agentName := ctx.Params(":agent")
	agentVer := ctx.ParamsInt64(":ver")
	//TODO: add auth
	owner := ctx.OrgId
	log.Debug("socket: agent name %s", agentName)
	log.Debug("socket: agent ver %d", agentVer)
	log.Debug("socket: agent orgid %d", owner)
	agent, err := connectedAgent(agentName, owner)
	if err != nil {
		taskServerAgentConnectionsFailedCount.Inc()
		log.Debug("socket: agent cant connect. %s", err)
		ctx.JSON(400, err.Error())
		return
	}

	c, err := upgrader.Upgrade(ctx.Resp, ctx.Req.Request, nil)
	if err != nil {
		log.Error(3, "socket: upgrade:", err)
		return
	}

	log.Debug("socket: agent %s connected.", agent.Name)

	sess := agent_session.NewSession(agent, agentVer, c)
	ActiveSockets.NewSocket(sess)
	sess.Start()
	//block until connection closes.
	<-sess.Done
	ActiveSockets.DeleteSocket(sess)
}
