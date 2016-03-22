package sqlstore

import (
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
			agentsById[r.Agent.Id] = &model.AgentDTO{
				Id:            r.Agent.Id,
				Name:          r.Agent.Name,
				Enabled:       r.Agent.Enabled,
				EnabledChange: r.Agent.EnabledChange,
				Owner:         r.Agent.Owner,
				Public:        r.Agent.Public,
				Online:        r.Agent.Online,
				OnlineChange:  r.Agent.OnlineChange,
				Created:       r.Agent.Created,
				Updated:       r.Agent.Updated,
				Tags:          []string{r.AgentTag.Tag},
			}
		} else {
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
	if query.Name != "" {
		sess.Where("agent.name = ?", query.Name)
	}
	if query.Enabled != "" {
		enabled, err := strconv.ParseBool(query.Enabled)
		if err != nil {
			return nil, err
		}
		sess.Where("agent.enabled=?", enabled)
	}
	if query.Public != "" {
		public, err := strconv.ParseBool(query.Public)
		if err != nil {
			return nil, err
		}
		sess.Where("agent.public=?", public)
	}
	if query.Tag != "" {
		sess.Join("INNER", []string{"agent_tag", "at"}, "agent.id = at.agent_id").Where("at.tag=?", query.Tag)
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
	sess.Asc(query.OrderBy).Limit(query.Limit, (query.Page-1)*query.Limit)
	err := sess.Join("LEFT", "agent_tag", "agent.id = agent_tag.agent_id").Find(&a)
	if err != nil {
		return nil, err
	}
	return a.ToAgentDTO(), nil
}

func GetAgentById(id int64, owner int64) (*model.AgentDTO, error) {
	sess, err := newSession(false, "agent")
	if err != nil {
		return nil, err
	}
	return getAgentById(sess, id, owner)
}

func getAgentById(sess *session, id int64, owner int64) (*model.AgentDTO, error) {
	var a agentWithTags
	err := sess.Where("agent.id=?", id).Join("INNER", "agent_tag", "agent.id = agent_tag.agent_id").Find(&a)
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
		Owner:         a.Owner,
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
			Owner:   a.Owner,
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
	existing, err := getAgentById(sess, a.Id, a.Owner)
	if err != nil {
		return err
	}
	if existing == nil {
		return model.AgentNotFound
	}
	// If the Owner is different, the only changes that can be made is to Tags.
	if a.Owner == existing.Owner {
		enabledChange := existing.EnabledChange
		if existing.Enabled != a.Enabled {
			enabledChange = time.Now()
		}
		agent := &model.Agent{
			Id:            a.Id,
			Name:          a.Name,
			Enabled:       a.Enabled,
			EnabledChange: enabledChange,
			Owner:         a.Owner,
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
		rawParams = append(rawParams, a.Id, a.Owner)
		p := make([]string, len(tagsToDelete))
		for i, t := range tagsToDelete {
			p[i] = "?"
			rawParams = append(rawParams, t)
		}
		rawSql := fmt.Sprintf("DELETE FROM agent_tag WHERE agent_id=? AND owner=? AND tag IN (%s)", strings.Join(p, ","))
		if _, err := sess.Exec(rawSql, rawParams...); err != nil {
			return err
		}
	}
	if len(tagsToAdd) > 0 {
		newAgentTags := make([]model.AgentTag, len(tagsToAdd))
		for i, tag := range tagsToAdd {
			newAgentTags[i] = model.AgentTag{
				Owner:   a.Owner,
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

func GetAgentsForTask(task *model.TaskDTO) ([]*AgentId, error) {
	sess, err := newSession(true, "agent")
	if err != nil {
		return nil, err
	}
	defer sess.Cleanup()
	agents, err := getAgentsForTask(sess, task)
	if err != nil {
		return nil, err
	}
	sess.Complete()
	return agents, nil
}

func getAgentsForTask(sess *session, t *model.TaskDTO) ([]*AgentId, error) {
	agents := make([]*AgentId, 0)
	switch t.Route.Type {
	case model.RouteAny:
		agents = append(agents, &AgentId{Id: t.Route.Config["id"].(int64)})
		return agents, nil
	case model.RouteByTags:
		tags := make([]string, len(t.Route.Config["tags"].([]string)))
		for i, tag := range t.Route.Config["tags"].([]string) {
			tags[i] = tag
		}
		sess.Join("LEFT", "agent_tag", "agent.id = agent_tag.agent_id")
		sess.Where("agent_tag.owner = ?", t.Owner)
		sess.In("agent_tag.tag", tags)
		sess.Cols("agent.id")
		err := sess.Find(&agents)
		return agents, err
	case model.RouteByIds:
		agents := make([]*AgentId, len(t.Route.Config["ids"].([]int64)))
		for i, id := range t.Route.Config["ids"].([]int64) {
			agents[i] = &AgentId{Id: id}
		}
		return agents, nil
	default:
		return nil, fmt.Errorf("unknown routeType")
	}
}

func DeleteAgent(id int64, owner int64) error {
	sess, err := newSession(true, "agent")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	err = deleteAgent(sess, id, owner)
	if err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func deleteAgent(sess *session, id int64, owner int64) error {
	existing, err := getAgentById(sess, id, owner)
	if err != nil {
		return err
	}
	rawSql := "DELETE FROM agent WHERE id=? and owner=?"
	if _, err := sess.Exec(rawSql, existing.Id, existing.Owner); err != nil {
		return err
	}
	rawSql = "DELETE FROM agent_tag WHERE agent_id=? and owner=?"
	if _, err := sess.Exec(rawSql, existing.Id, existing.Owner); err != nil {
		return err
	}
	rawSql = "DELETE FROM agent_metric WHERE agent_id=?"
	if _, err := sess.Exec(rawSql, existing.Id); err != nil {
		return err
	}
	rawSql = "DELETE FROM route_by_id_index WHERE agent_id=?"
	if _, err := sess.Exec(rawSql, existing.Id); err != nil {
		return err
	}
	return nil
}
