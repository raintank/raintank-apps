package api

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/Unknwon/macaron"
	"github.com/gorilla/websocket"

	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/server/agent_session"
	"github.com/raintank/raintank-apps/server/model"
	"github.com/raintank/raintank-apps/server/sqlstore"
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
	s.Unlock()
}

func (s *socketList) EmitTask(task *model.TaskDTO, event string) error {
	agents, err := sqlstore.GetAgentsForTask(task)
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
	s.Lock()
	for _, agent := range agents {
		if as, ok := s.Sockets[agent.Id]; ok {
			log.Debugf("sending %s event to agent %d", event, agent.Id)
			as.SocketSession.Emit(e)
		}
	}
	s.Unlock()
	return nil
}

func (s *socketList) NewSocket(a *agent_session.AgentSession) {
	s.Lock()
	existing, ok := s.Sockets[a.Agent.Id]
	if ok {
		log.Debugf("new connection for agent %d - %s, closing existing session", a.Agent.Id, a.Agent.Name)
		existing.Close()
	}
	s.Sockets[a.Agent.Id] = a
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
	ActiveSockets.NewSocket(sess)
	sess.Start()
	//block until connection closes.
	<-sess.Done
}
