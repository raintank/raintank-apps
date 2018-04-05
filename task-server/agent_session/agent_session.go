package agent_session

import (
	"encoding/json"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/pkg/session"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
	"github.com/raintank/worldping-api/pkg/log"
)

type AgentSession struct {
	Agent         *model.AgentDTO
	AgentVersion  int64
	dbSession     *model.AgentSession
	SocketSession *session.Session
	Done          chan struct{}
	Shutdown      chan struct{}
	closing       bool
}

func NewSession(agent *model.AgentDTO, agentVer int64, conn *websocket.Conn) *AgentSession {
	a := &AgentSession{
		Agent:         agent,
		AgentVersion:  agentVer,
		Done:          make(chan struct{}),
		Shutdown:      make(chan struct{}),
		SocketSession: session.NewSession(conn, 10),
	}
	return a
}

func (a *AgentSession) Start() error {
	if err := a.saveDbSession(); err != nil {
		log.Error(3, "unable to add agentSession to DB. %s", err.Error())
		a.close()
		return err
	}

	log.Debug("setting handler for disconnect event.")
	if err := a.SocketSession.On("disconnect", a.OnDisconnect()); err != nil {
		log.Error(3, "failed to bind disconnect event. %s", err.Error())
		a.close()
		return err
	}

	log.Info("starting session %s", a.SocketSession.Id)
	go a.SocketSession.Start()

	// run background tasks for this session.
	go a.sendHeartbeat()
	go a.sendTaskListPeriodically()
	a.sendTaskList()
	return nil
}

func (a *AgentSession) Close() {
	a.close()
}

func (a *AgentSession) close() {
	if !a.closing {
		a.closing = true
		close(a.Shutdown)
		log.Debug("closing websocket")
		a.SocketSession.Close()
		log.Debug("websocket closed")

		a.cleanup()
		close(a.Done)
	}
}

func (a *AgentSession) saveDbSession() error {
	host, _ := os.Hostname()
	dbSess := &model.AgentSession{
		Id:       a.SocketSession.Id,
		AgentId:  a.Agent.Id,
		Version:  a.AgentVersion,
		RemoteIp: a.SocketSession.Conn.RemoteAddr().String(),
		Server:   host,
		Created:  time.Now(),
	}
	err := sqlstore.AddAgentSession(dbSess)
	if err != nil {
		return err
	}
	a.dbSession = dbSess
	return nil
}

func (a *AgentSession) cleanup() {
	//remove agentSession from DB.
	if a.dbSession != nil {
		log.Debug("deleting agent_session for %s from DB", a.Agent.Name)
		sqlstore.DeleteAgentSession(a.dbSession)
	} else {
		log.Debug("agent_session for %s has no db session.", a.Agent.Name)
	}
}

func (a *AgentSession) OnDisconnect() interface{} {
	return func() {
		log.Debug("session %s has disconnected", a.SocketSession.Id)
		a.close()
	}
}

func (a *AgentSession) sendHeartbeat() {
	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-a.Shutdown:
			log.Debug("session ended stopping heartbeat.")
			return
		case t := <-ticker.C:
			e := &message.Event{Event: "heartbeat", Payload: []byte(t.String())}
			err := a.SocketSession.Emit(e)
			if err != nil {
				log.Error(3, "failed to emit heartbeat event. %s", err)
			}
		}
	}
}

func (a *AgentSession) sendTaskListPeriodically() {
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-a.Shutdown:
			log.Debug("session ended stopping taskListPeriodically.")
			return
		case <-ticker.C:
			a.sendTaskList()
		}
	}
}

func (a *AgentSession) sendTaskList() {
	log.Debug("sending TaskUpdate to %s", a.SocketSession.Id)
	tasks, err := sqlstore.GetAgentTasks(a.Agent)
	if err != nil {
		log.Error(3, "failed to get task list. %s", err)
		return
	}
	body, err := json.Marshal(&tasks)
	if err != nil {
		log.Error(3, "failed to Marshal task list to json. %s", err)
		return
	}
	e := &message.Event{Event: "taskList", Payload: body}
	err = a.SocketSession.Emit(e)
	if err != nil {
		log.Error(3, "failed to emit taskList event. %s", err)
	}
}
