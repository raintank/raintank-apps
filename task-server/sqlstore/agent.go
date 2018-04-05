package sqlstore

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/raintank/raintank-apps/task-server/model"
)

type agentWithTag struct {
	model.Agent    `xorm:"extends"`
	model.AgentTag `xorm:"extends"`
}

type agentWithTags []*agentWithTag

func (agentWithTags) TableName() string {
	return "agent"
}

func (rows agentWithTags) ToAgentDTO() []*model.AgentDTO {
	agentsById := make(map[int64]*model.AgentDTO)
	for _, r := range rows {
		a, ok := agentsById[r.Agent.Id]
		if !ok {
			tags := make([]string, 0)
			if r.AgentTag.Tag != "" {
				tags = append(tags, r.AgentTag.Tag)
			}
			agentsById[r.Agent.Id] = &model.AgentDTO{
				Id:            r.Agent.Id,
				Name:          r.Agent.Name,
				Enabled:       r.Agent.Enabled,
				EnabledChange: r.Agent.EnabledChange,
				OrgId:         r.Agent.OrgId,
				Public:        r.Agent.Public,
				Online:        r.Agent.Online,
				OnlineChange:  r.Agent.OnlineChange,
				Created:       r.Agent.Created,
				Updated:       r.Agent.Updated,
				Tags:          tags,
			}
		} else if r.Tag != "" {
			a.Tags = append(a.Tags, r.Tag)
		}
	}
	agents := make([]*model.AgentDTO, len(agentsById))
	i := 0
	for _, a := range agentsById {
		agents[i] = a
		i++
	}
	return agents
}

func GetAgents(query *model.GetAgentsQuery) ([]*model.AgentDTO, error) {
	sess, err := newSession(false, "agent")
	if err != nil {
		return nil, err
	}
	return getAgents(sess, query)
}

func getAgents(sess *session, query *model.GetAgentsQuery) ([]*model.AgentDTO, error) {
	var a agentWithTags
	var rawSQL bytes.Buffer
	args := make([]interface{}, 0)

	var where bytes.Buffer
	whereArgs := make([]interface{}, 0)
	prefix := "WHERE"

	fmt.Fprint(&rawSQL, "SELECT agent.*, agent_tag.* FROM agent LEFT JOIN agent_tag ON  agent.id = agent_tag.agent_id ")
	if len(query.Tag) > 0 {
		fmt.Fprint(&rawSQL, "INNER JOIN agent_tag as at ON agent.id = at.agent_id ")
		p := make([]string, len(query.Tag))
		for i, tag := range query.Tag {
			p[i] = "?"
			whereArgs = append(whereArgs, tag)
		}
		filter := fmt.Sprintf("at.tag IN (%s)", strings.Join(p, ","))

		fmt.Fprintf(&where, "%s %s ", prefix, filter)
		prefix = "AND"
	}

	if query.Name != "" {
		fmt.Fprintf(&where, "%s agent.name=? ", prefix)
		whereArgs = append(whereArgs, query.Name)
		prefix = "AND"
	}
	if query.Enabled != "" {
		enabled, err := strconv.ParseBool(query.Enabled)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&where, "%s agent.enabled=? ", prefix)
		whereArgs = append(whereArgs, enabled)
		prefix = "AND"
	}
	if query.Public != "" {
		public, err := strconv.ParseBool(query.Public)
		if err != nil {
			return nil, err
		}
		if public {
			fmt.Fprintf(&where, "%s agent.public=1 ", prefix)
			prefix = "AND"
		} else {
			fmt.Fprintf(&where, "%s (agent.public=0 AND agent.org_id=?) ", prefix)
			whereArgs = append(whereArgs, query.OrgId)
			prefix = "AND"
		}
	} else {
		fmt.Fprintf(&where, "%s (agent.org_id=? OR agent.public=1) ", prefix)
		whereArgs = append(whereArgs, query.OrgId)
		prefix = "AND"
	}

	if query.OrderBy == "" {
		query.OrderBy = "name"
	}
	if query.Limit == 0 {
		query.Limit = 50
	}
	if query.Page == 0 {
		query.Page = 1
	}

	fmt.Fprint(&rawSQL, where.String())
	args = append(args, whereArgs...)
	fmt.Fprintf(&rawSQL, "ORDER BY `%s` ASC LIMIT %d, %d", query.OrderBy, (query.Page-1)*query.Limit, query.Limit)

	err := sess.Sql(rawSQL.String(), args...).Find(&a)

	if err != nil {
		return nil, err
	}
	return a.ToAgentDTO(), nil
}

func onlineAgentsWithNoSession(sess *session) ([]*model.AgentDTO, error) {
	sess.Table("agent")
	sess.Join("LEFT", "agent_tag", "agent.id=agent_tag.agent_id")
	sess.Join("LEFT", "agent_session", "agent.id = agent_session.agent_id")
	sess.Where("agent.online=1").And("agent_session.id is NULL")
	sess.Cols("`agent`.*", "`agent_tag`.*")
	var a agentWithTags
	err := sess.Find(&a)
	if err != nil {
		return nil, err
	}

	return a.ToAgentDTO(), nil
}

func GetAgentById(id int64, orgId int64) (*model.AgentDTO, error) {
	sess, err := newSession(false, "agent")
	if err != nil {
		return nil, err
	}
	return getAgentById(sess, id, orgId)
}

func getAgentById(sess *session, id int64, orgId int64) (*model.AgentDTO, error) {
	var a agentWithTags
	sess.Where("agent.id=?", id)
	if orgId != 0 {
		sess.And("agent.org_id=?", orgId)
	}
	err := sess.Join("LEFT", "agent_tag", "agent.id = agent_tag.agent_id").Find(&a)
	if err != nil {
		return nil, err
	}
	if len(a) == 0 {
		return nil, model.AgentNotFound
	}
	return a.ToAgentDTO()[0], nil
}

func AddAgent(a *model.AgentDTO) error {
	sess, err := newSession(true, "agent")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = addAgent(sess, a); err != nil {
		return err
	}
	sess.Complete()
	return nil

}

func addAgent(sess *session, a *model.AgentDTO) error {
	agent := &model.Agent{
		Name:          a.Name,
		Enabled:       a.Enabled,
		EnabledChange: time.Now(),
		OrgId:         a.OrgId,
		Public:        a.Public,
		Online:        false,
		OnlineChange:  time.Now(),
		Created:       time.Now(),
		Updated:       time.Now(),
	}

	sess.UseBool("public")
	sess.UseBool("enabled")
	sess.UseBool("online")
	if _, err := sess.Insert(agent); err != nil {
		return err
	}
	a.Id = agent.Id
	a.Created = agent.Created
	a.Updated = agent.Updated

	agentTags := make([]model.AgentTag, 0, len(a.Tags))
	for _, tag := range a.Tags {
		agentTags = append(agentTags, model.AgentTag{
			OrgId:   a.OrgId,
			AgentId: agent.Id,
			Tag:     tag,
			Created: time.Now(),
		})
	}
	if len(agentTags) > 0 {
		sess.Table("agent_tag")
		if _, err := sess.Insert(&agentTags); err != nil {
			return err
		}
	}
	return nil
}

func UpdateAgent(a *model.AgentDTO) error {
	sess, err := newSession(true, "agent")
	if err != nil {
		return err
	}
	defer sess.Cleanup()

	err = updateAgent(sess, a)
	if err != nil {
		return err
	}
	sess.Complete()
	return err
}

func updateAgent(sess *session, a *model.AgentDTO) error {
	existing, err := getAgentById(sess, a.Id, 0)
	if err != nil {
		return err
	}
	if existing == nil || (a.OrgId != existing.OrgId && !existing.Public) {
		return model.AgentNotFound
	}
	// If the OrgId is different, the only changes that can be made is to Tags.
	if a.OrgId == existing.OrgId {
		enabledChange := existing.EnabledChange
		if existing.Enabled != a.Enabled {
			enabledChange = time.Now()
		}
		agent := &model.Agent{
			Id:            a.Id,
			Name:          a.Name,
			Enabled:       a.Enabled,
			EnabledChange: enabledChange,
			OrgId:         a.OrgId,
			Public:        a.Public,
			Created:       a.Created,
			Updated:       time.Now(),
		}
		sess.UseBool("public")
		sess.UseBool("enabled")
		if _, err := sess.Id(agent.Id).Update(agent); err != nil {
			return err
		}
		a.Updated = agent.Updated
	}

	tagMap := make(map[string]bool)
	tagsToDelete := make([]string, 0)
	tagsToAddMap := make(map[string]bool, 0)
	// create map of current tags
	for _, t := range existing.Tags {
		tagMap[t] = false
	}

	// create map of tags to add. We use a map
	// to ensure that we only add each tag once.
	for _, t := range a.Tags {
		if _, ok := tagMap[t]; !ok {
			tagsToAddMap[t] = true
		}
		// mark that this tag has been seen.
		tagMap[t] = true
	}

	//create list of tags to delete
	for t, seen := range tagMap {
		if !seen {
			tagsToDelete = append(tagsToDelete, t)
		}
	}

	// create list of tags to add.
	tagsToAdd := make([]string, len(tagsToAddMap))
	i := 0
	for t := range tagsToAddMap {
		tagsToAdd[i] = t
		i += 1
	}
	if len(tagsToDelete) > 0 {
		rawParams := make([]interface{}, 0)
		rawParams = append(rawParams, a.Id, a.OrgId)
		p := make([]string, len(tagsToDelete))
		for i, t := range tagsToDelete {
			p[i] = "?"
			rawParams = append(rawParams, t)
		}
		rawSql := fmt.Sprintf("DELETE FROM agent_tag WHERE agent_id=? AND org_id=? AND tag IN (%s)", strings.Join(p, ","))
		if _, err := sess.Exec(rawSql, rawParams...); err != nil {
			return err
		}
	}
	if len(tagsToAdd) > 0 {
		newAgentTags := make([]model.AgentTag, len(tagsToAdd))
		for i, tag := range tagsToAdd {
			newAgentTags[i] = model.AgentTag{
				OrgId:   a.OrgId,
				AgentId: a.Id,
				Tag:     tag,
				Created: time.Now(),
			}
		}
		sess.Table("agent_tag")
		if _, err := sess.Insert(&newAgentTags); err != nil {
			return err
		}
	}

	return nil
}

type AgentId struct {
	Id int64
}

func GetAgentsForTask(task *model.TaskDTO) ([]int64, error) {
	sess, err := newSession(false, "agent")
	if err != nil {
		return nil, err
	}

	return getAgentsForTask(sess, task)
}

func getAgentsForTask(sess *session, t *model.TaskDTO) ([]int64, error) {
	agents := make([]*AgentId, 0)
	switch t.Route.Type {
	case model.RouteAny:
		err := sess.Sql("SELECT agent_id as id FROM route_by_any_index where task_id=?", t.Id).Find(&agents)
		if err != nil {
			return nil, err
		}
	case model.RouteByTags:
		tags := make([]string, len(t.Route.Config["tags"].([]string)))
		for i, tag := range t.Route.Config["tags"].([]string) {
			tags[i] = tag
		}
		sess.Join("LEFT", "agent_tag", "agent.id = agent_tag.agent_id")
		sess.Where("agent_tag.org_id = ?", t.OrgId)
		sess.In("agent_tag.tag", tags)
		sess.Cols("agent.id")
		err := sess.Find(&agents)
		if err != nil {
			return nil, err
		}
	case model.RouteByIds:
		for _, id := range t.Route.Config["ids"].([]int64) {
			agents = append(agents, &AgentId{Id: id})
		}
	default:
		return nil, fmt.Errorf("unknown routeType")
	}
	agentIds := make([]int64, len(agents))
	for i, a := range agents {
		agentIds[i] = a.Id
	}
	return agentIds, nil
}

func DeleteAgent(id int64, orgId int64) error {
	sess, err := newSession(true, "agent")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	err = deleteAgent(sess, id, orgId)
	if err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func deleteAgent(sess *session, id int64, orgId int64) error {
	existing, err := getAgentById(sess, id, orgId)
	if err != nil {
		return err
	}
	rawSql := "DELETE FROM agent WHERE id=? and org_id=?"
	if _, err := sess.Exec(rawSql, existing.Id, existing.OrgId); err != nil {
		return err
	}
	rawSql = "DELETE FROM agent_tag WHERE agent_id=? and org_id=?"
	if _, err := sess.Exec(rawSql, existing.Id, existing.OrgId); err != nil {
		return err
	}
	rawSql = "DELETE FROM route_by_id_index WHERE agent_id=?"
	if _, err := sess.Exec(rawSql, existing.Id); err != nil {
		return err
	}
	return nil
}
