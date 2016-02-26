package sqlstore

import (
	"github.com/raintank/raintank-apps/server/model"
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
	return nil
}

func DeleteAgentSession(id string) error {
	sess, err := newSession(true, "agent_session")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = deleteAgentSession(sess, id); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func deleteAgentSession(sess *session, id string) error {
	var rawSql = "DELETE FROM agent_session WHERE id=?"
	_, err := sess.Exec(rawSql, id)
	if err != nil {
		return err
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
	return nil
}
