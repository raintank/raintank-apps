package sqlstore

import (
	"github.com/raintank/raintank-apps/server/model"
)

func AddAgentSession(a *model.AgentSession) (*model.AgentSession, error) {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return nil, err
	}
	return addAgentSession(sess, a)
}

func addAgentSession(sess *session, a *model.AgentSession) (*model.AgentSession, error) {
	defer sess.Cleanup()
	if _, err := sess.Insert(a); err != nil {
		return nil, err
	}
	sess.Complete()
	return a, nil
}

func DeleteAgentSession(id string) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	return deleteAgentSession(sess, id)
}

func deleteAgentSession(sess *session, id string) error {
	defer sess.Cleanup()
	var rawSql = "DELETE FROM agent_session WHERE id=?"
	_, err := sess.Exec(rawSql, id)
	if err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func DeleteAgentSessionsByServer(server string) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	return deleteAgentSessionsByServer(sess, server)
}

func deleteAgentSessionsByServer(sess *session, server string) error {
	defer sess.Cleanup()
	var rawSql = "DELETE FROM agent_session WHERE server=?"
	_, err := sess.Exec(rawSql, server)
	if err != nil {
		return err
	}
	sess.Complete()
	return nil
}
