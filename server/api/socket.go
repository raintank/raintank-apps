package api

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Unknwon/macaron"
	"github.com/gorilla/websocket"
	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/pkg/session"

	"github.com/raintank/raintank-apps/pkg/model"
	"github.com/raintank/raintank-apps/server/sqlstore"
)

var upgrader = websocket.Upgrader{} // use default options

func authRequest(agentName string) (*model.AgentDTO, error) {
	if agentName == "" {
		return nil, errors.New("agent name not specified.")
	}
	q := model.GetAgentsQuery{
		Name:  agentName,
		Owner: "admin",
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
	agent, err := authRequest(agentName)
	if err != nil {
		ctx.JSON(400, err.Error())
		return
	}
	log.Debugf("agent %s connected.", agent.Name)

	c, err := upgrader.Upgrade(ctx.Resp, ctx.Req.Request, nil)

	if err != nil {
		log.Errorf("upgrade:", err)
		return
	}
	sess := session.NewSession(c, 10)
	log.Infof("session %s has connected", sess.Id)
	done := make(chan struct{})
	if err = sess.On("disconnect", func() {
		log.Debugf("session %s has disconnected", sess.Id)
		close(done)
	}); err != nil {
		log.Errorf("failed to bind disconnect event. %s", err.Error())
		sess.Close()
		return
	}

	log.Debug("setting handler for catalog event.")
	if err = sess.On("catalog", HandleCatalog(sess)); err != nil {
		log.Errorf("failed to bind catalog event handler. %s", err.Error())
		sess.Close()
		return
	}

	log.Infof("starting session %s", sess.Id)
	go sess.Start()

	// run background tasks for this session.
	go sendHeartbeat(done, sess)

	//block until connection closes.
	<-done
}

func sendHeartbeat(done chan struct{}, sess *session.Session) {
	ticker := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-done:
			log.Debug("session ended stopping heartbeat.")
			return
		case t := <-ticker.C:
			e := &message.Event{Event: "heartbeat", Payload: []byte(t.String())}
			err := sess.Emit(e)
			if err != nil {
				log.Error("failed to emit heartbeat event.")
			}
		}
	}
}

func HandleCatalog(sess *session.Session) interface{} {
	return func(body []byte) {
		catalog := make([]*rbody.Metric, 0)
		if err := json.Unmarshal(body, &catalog); err != nil {
			log.Error(err)
			return
		}
		log.Debugf("Received catalog for session %s: %s", sess.Id, body)
	}
}
