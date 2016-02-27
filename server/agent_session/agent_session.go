package agent_session

import (
	"encoding/json"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/op/go-logging"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/pkg/session"
	"github.com/raintank/raintank-apps/server/model"
	"github.com/raintank/raintank-apps/server/sqlstore"
)

var log = logging.MustGetLogger("default")

type AgentSession struct {
	Agent         *model.AgentDTO
	AgentVersion  int64
	dbSession     *model.AgentSession
	SocketSession *session.Session
	Done          chan struct{}
}

func NewSession(agent *model.AgentDTO, agentVer int64, conn *websocket.Conn) *AgentSession {
	a := &AgentSession{
		Agent:         agent,
		AgentVersion:  agentVer,
		Done:          make(chan struct{}),
		SocketSession: session.NewSession(conn, 10),
	}
	return a
}

func (a *AgentSession) Start() error {
	if err := a.saveDbSession(); err != nil {
		log.Errorf("unable to add agentSession to DB. %s", err.Error())
		a.handleError(err)
		return err
	}

	log.Debug("setting handler for disconnect event.")
	if err := a.SocketSession.On("disconnect", a.OnDisconnect()); err != nil {
		log.Errorf("failed to bind disconnect event. %s", err.Error())
		a.handleError(err)
		return err
	}

	log.Debug("setting handler for catalog event.")
	if err := a.SocketSession.On("catalog", a.HandleCatalog()); err != nil {
		log.Errorf("failed to bind catalog event handler. %s", err.Error())
		a.handleError(err)
		return err
	}

	log.Infof("starting session %s", a.SocketSession.Id)
	go a.SocketSession.Start()

	// run background tasks for this session.
	go a.sendHeartbeat()
	return nil
}

func (a *AgentSession) handleError(err error) {
	a.SocketSession.Close()
	close(a.Done)
	a.cleanup()
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
		sqlstore.DeleteAgentSession(a.dbSession.Id)
	}
}

func (a *AgentSession) OnDisconnect() interface{} {
	return func() {
		log.Debugf("session %s has disconnected", a.SocketSession.Id)

		//TODO: delete AgentSession from DB.
		close(a.Done)
		a.cleanup()
	}
}

func (a *AgentSession) HandleCatalog() interface{} {
	return func(body []byte) {
		catalog := make([]*rbody.Metric, 0)
		if err := json.Unmarshal(body, &catalog); err != nil {
			log.Error(err)
			return
		}
		log.Debugf("Received catalog for session %s: %s", a.SocketSession.Id, body)
		metrics := make([]*model.Metric, len(catalog))
		for i, m := range catalog {
			metrics[i] = &model.Metric{
				Owner:     a.Agent.Owner,
				Public:    a.Agent.Public,
				Namespace: m.Namespace,
				Version:   int64(m.Version),
				Policy:    m.Policy,
			}
		}
		err := sqlstore.AddMissingMetrics(metrics)
		if err != nil {
			log.Errorf("failed to update metrics in DB. %s", err)
		}
	}
}

func (a *AgentSession) sendHeartbeat() {
	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-a.Done:
			log.Debug("session ended stopping heartbeat.")
			return
		case t := <-ticker.C:
			e := &message.Event{Event: "heartbeat", Payload: []byte(t.String())}
			err := a.SocketSession.Emit(e)
			if err != nil {
				log.Error("failed to emit heartbeat event.")
			}
		}
	}
}
