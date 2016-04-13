package sqlstore

import (
	"time"

	"github.com/raintank/raintank-apps/task-server/model"
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

func DeleteAgentSession(a *model.AgentSession) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = deleteAgentSession(sess, a); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func deleteAgentSession(sess *session, a *model.AgentSession) error {
	var rawSql = "DELETE FROM agent_session WHERE id=?"
	_, err := sess.Exec(rawSql, a.Id)
	if err != nil {
		return err
	}
	total, err := sess.Where("agent_session.agent_id=?", a.AgentId).Count(&model.AgentSession{})
	if err != nil {
		return err
	}
	if total == 0 {
		rawSql := "UPDATE agent set online=0, online_change=? where id=?"
		_, err := sess.Exec(rawSql, a.AgentId, time.Now())
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteAgentSessionsByServer(server string) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = deleteAgentSessionsByServer(sess, server); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func deleteAgentSessionsByServer(sess *session, server string) error {
	var rawSql = "DELETE FROM agent_session WHERE server=?"
	_, err := sess.Exec(rawSql, server)
	if err != nil {
		return err
	}

	// set the online state for all agents that now have no sessions.
	rawSql = `UPDATE agent set online=0, online_change=? where id IN (SELECT t.id 
	FROM (SELECT id from agent WHERE agent.online=1) as t LEFT JOIN agent_session ON t.id = agent_session.agent_id
	WHERE agent_session.id is NULL)`
	_, err = sess.Exec(rawSql, time.Now())
	if err != nil {
		return err
	}
	return nil
}

func GetAgentSessionsByServer(server string) ([]*model.AgentSession, error) {
	sess, err := newSession(false, "agent_session")
	if err != nil {
		return nil, err
	}
	return getAgentSessionsByServer(sess, server)
}

func getAgentSessionsByServer(sess *session, server string) ([]*model.AgentSession, error) {
	agentSessions := make([]*model.AgentSession, 0)
	err := sess.Where("server=?", server).Find(&agentSessions)
	if err != nil {
		return nil, err
	}
	return agentSessions, nil
}

func GetAgentSessionsByAgentId(agentId int64) ([]*model.AgentSession, error) {
	sess, err := newSession(false, "agent_session")
	if err != nil {
		return nil, err
	}
	return getAgentSessionsByAgentId(sess, agentId)
}

func getAgentSessionsByAgentId(sess *session, agentId int64) ([]*model.AgentSession, error) {
	agentSessions := make([]*model.AgentSession, 0)
	err := sess.Where("agent_id=?", agentId).Find(&agentSessions)
	if err != nil {
		return nil, err
	}
	return agentSessions, nil
}
