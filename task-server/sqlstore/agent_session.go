package sqlstore

import (
	"time"

	"github.com/raintank/raintank-apps/task-server/event"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/worldping-api/pkg/log"
)

func AddAgentSession(a *model.AgentSession) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()

	if err := addAgentSession(sess, a); err != nil {
		return err
	}
	sess.Complete()
	return err
}

func addAgentSession(sess *session, a *model.AgentSession) error {
	if _, err := sess.Insert(a); err != nil {
		return err
	}
	// set Agent state to online.
	rawSql := "UPDATE agent set online=1, online_change=? where id=?"
	_, err := sess.Exec(rawSql, time.Now(), a.AgentId)
	if err != nil {
		return err
	}
	return nil
}

func AgentSessionHeartbeat(a *model.AgentSession) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()

	if err := agentSessionHeartbeat(sess, a); err != nil {
		return err
	}
	sess.Complete()
	return err
}
func agentSessionHeartbeat(sess *session, a *model.AgentSession) error {
	rawSql := "UPDATE agent_session set heartbeat=Now() where id=?"
	_, err := sess.Exec(rawSql, a.Id)
	if err != nil {
		return err
	}
	return nil
}

func DeleteAgentSessionsWithStaleHeartbeat(stale time.Duration) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	events, err := deleteAgentSessionsWithStaleHeartbeat(sess, stale)
	if err != nil {
		return err
	}
	sess.Complete()
	for _, e := range events {
		event.Publish(e, 0)
	}
	return nil
}

func deleteAgentSessionsWithStaleHeartbeat(sess *session, stale time.Duration) ([]event.Event, error) {
	events := make([]event.Event, 0)
	var rawSql = "DELETE FROM agent_session WHERE heartbeat < ?"
	_, err := sess.Exec(rawSql, time.Now().Add(-1*stale))
	if err != nil {
		return nil, err
	}

	// Get agents that are now offline.
	nowOffline, err := onlineAgentsWithNoSession(sess)
	if err != nil {
		return nil, err
	}
	if len(nowOffline) > 0 {
		agentIds := make([]int64, len(nowOffline))
		for i, a := range nowOffline {
			a.Online = false
			a.OnlineChange = time.Now()
			agentIds[i] = a.Id
			log.Info("Agent %s has no sessions. Marking as offline.", a.Name)
		}
		sess.UseBool("online")
		update := map[string]interface{}{"online": false, "online_change": time.Now()}
		_, err = sess.Table(&model.Agent{}).In("id", agentIds).Update(update)
		if err != nil {
			return nil, err
		}
		for _, a := range nowOffline {
			events = append(events, &event.AgentOffline{Ts: time.Now(), Payload: a})
		}
	}
	return events, nil
}

func DeleteAgentSession(a *model.AgentSession) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	events, err := deleteAgentSession(sess, a)
	if err != nil {
		return err
	}
	sess.Complete()
	for _, e := range events {
		event.Publish(e, 0)
	}
	return nil
}

func deleteAgentSession(sess *session, a *model.AgentSession) ([]event.Event, error) {
	events := make([]event.Event, 0)
	var rawSql = "DELETE FROM agent_session WHERE id=?"
	_, err := sess.Exec(rawSql, a.Id)
	if err != nil {
		return nil, err
	}

	// we query here to prevent race conditions when agents dicsonnect from one task-server node
	// and connect to another.  The new connection may establish before the old connection times out.
	total, err := sess.Where("agent_session.agent_id=?", a.AgentId).Count(&model.AgentSession{})
	if err != nil {
		return nil, err
	}
	if total == 0 {
		agent, err := getAgentById(sess, a.AgentId, 0)
		if err != nil {
			return nil, err
		}
		log.Info("Agent %s has no sessions. Marking as offline.", agent.Name)
		agent.Online = false
		agent.OnlineChange = time.Now()
		sess.UseBool("online")
		_, err = sess.Id(agent.Id).Update(agent)
		if err != nil {
			return nil, err
		}
		events = append(events, &event.AgentOffline{Ts: time.Now(), Payload: agent})
	}
	return events, nil
}

func DeleteAgentSessionsByServer(server string) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	events, err := deleteAgentSessionsByServer(sess, server)
	if err != nil {
		return err
	}
	sess.Complete()
	for _, e := range events {
		event.Publish(e, 0)
	}
	return nil
}

func deleteAgentSessionsByServer(sess *session, server string) ([]event.Event, error) {
	events := make([]event.Event, 0)
	var rawSql = "DELETE FROM agent_session WHERE server=?"
	_, err := sess.Exec(rawSql, server)
	if err != nil {
		return nil, err
	}

	// Get agents that are now offline.
	nowOffline, err := onlineAgentsWithNoSession(sess)
	if err != nil {
		return nil, err
	}
	if len(nowOffline) > 0 {
		agentIds := make([]int64, len(nowOffline))
		for i, a := range nowOffline {
			a.Online = false
			a.OnlineChange = time.Now()
			agentIds[i] = a.Id
			log.Info("Agent %s has no sessions. Marking as offline.", a.Name)
		}
		sess.UseBool("online")
		update := map[string]interface{}{"online": false, "online_change": time.Now()}
		_, err = sess.Table(&model.Agent{}).In("id", agentIds).Update(update)
		if err != nil {
			return nil, err
		}
		for _, a := range nowOffline {
			events = append(events, &event.AgentOffline{Ts: time.Now(), Payload: a})
		}
	}
	return events, nil
}

func GetAgentSessionsByServer(server string) ([]model.AgentSession, error) {
	sess, err := newSession(false, "agent_session")
	if err != nil {
		return nil, err
	}
	return getAgentSessionsByServer(sess, server)
}

func getAgentSessionsByServer(sess *session, server string) ([]model.AgentSession, error) {
	agentSessions := make([]model.AgentSession, 0)
	err := sess.Where("server=?", server).Find(&agentSessions)
	if err != nil {
		return nil, err
	}
	return agentSessions, nil
}

func GetAgentSessionsByAgentId(agentId int64) ([]model.AgentSession, error) {
	sess, err := newSession(false, "agent_session")
	if err != nil {
		return nil, err
	}
	return getAgentSessionsByAgentId(sess, agentId)
}

func getAgentSessionsByAgentId(sess *session, agentId int64) ([]model.AgentSession, error) {
	agentSessions := make([]model.AgentSession, 0)
	err := sess.Where("agent_id=?", agentId).Find(&agentSessions)
	if err != nil {
		return nil, err
	}
	return agentSessions, nil
}
