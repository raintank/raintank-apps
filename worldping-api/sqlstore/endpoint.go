package sqlstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raintank/raintank-apps/worldping-api/model"
)

type endpointRow struct {
	model.Endpoint    `xorm:"extends"`
	model.Check       `xorm:"extends"`
	model.EndpointTag `xorm:"extends"`
}

type endpointRows []*endpointRow

func (endpointRows) TableName() string {
	return "endpoint"
}

func (rows endpointRows) ToDTO() []*model.EndpointDTO {
	endpointsById := make(map[int64]*model.EndpointDTO)
	endpointChecksById := make(map[int64]map[int64]*model.Check)
	endpointTagsById := make(map[int64]map[string]struct{})
	for _, r := range rows {
		_, ok := endpointsById[r.Endpoint.Id]

		check := &model.Check{
			Id:             r.Check.Id,
			OrgId:          r.Check.OrgId,
			EndpointId:     r.Check.EndpointId,
			Type:           r.Check.Type,
			Frequency:      r.Check.Frequency,
			Enabled:        r.Check.Enabled,
			State:          r.Check.State,
			StateCheck:     r.Check.StateCheck,
			StateChange:    r.Check.StateChange,
			Settings:       r.Check.Settings,
			HealthSettings: r.Check.HealthSettings,
			Created:        r.Check.Created,
			Updated:        r.Check.Updated,
			TaskId:         r.Check.TaskId,
		}
		if !ok {
			endpointsById[r.Endpoint.Id] = &model.EndpointDTO{
				Id:      r.Endpoint.Id,
				OrgId:   r.Endpoint.OrgId,
				Name:    r.Endpoint.Name,
				Slug:    r.Endpoint.Slug,
				Checks:  make([]*model.Check, 0),
				Tags:    make([]string, 0),
				Created: r.Endpoint.Created,
				Updated: r.Endpoint.Updated,
			}
			endpointChecksById[r.Endpoint.Id] = make(map[int64]*model.Check)
			endpointTagsById[r.Endpoint.Id] = make(map[string]struct{})
			if check.Id != 0 {
				endpointChecksById[r.Endpoint.Id][check.Id] = check
			}
			if r.EndpointTag.Tag != "" {
				endpointTagsById[r.Endpoint.Id][r.EndpointTag.Tag] = struct{}{}
			}
		} else {
			if check.Id != 0 {
				_, ecOk := endpointChecksById[r.Endpoint.Id][check.Id]
				if !ecOk {
					endpointChecksById[r.Endpoint.Id][check.Id] = check
				}
			}
			if r.EndpointTag.Tag != "" {
				_, tagOk := endpointTagsById[r.Endpoint.Id][r.EndpointTag.Tag]
				if !tagOk {
					endpointTagsById[r.Endpoint.Id][r.EndpointTag.Tag] = struct{}{}
				}
			}
		}
	}
	endpoints := make([]*model.EndpointDTO, len(endpointsById))
	i := 0
	for _, e := range endpointsById {
		for _, c := range endpointChecksById[e.Id] {
			e.Checks = append(e.Checks, c)
		}

		for t, _ := range endpointTagsById[e.Id] {
			e.Tags = append(e.Tags, t)
		}

		endpoints[i] = e
		i++
	}
	return endpoints
}

func GetEndpoints(query *model.GetEndpointsQuery) ([]*model.EndpointDTO, error) {
	sess, err := newSession(false, "endpoint")
	if err != nil {
		return nil, err
	}
	return getEndpoints(sess, query)
}

func getEndpoints(sess *session, query *model.GetEndpointsQuery) ([]*model.EndpointDTO, error) {
	var e endpointRows
	if query.Name != "" {
		sess.Where("endpoint.name like ?", query.Name)
	}
	if query.Tag != "" {
		sess.Join("INNER", []string{"endpoint_tag", "et"}, "endpoint.id = et.endpoint_id").Where("et.tag=?", query.Tag)
	}
	if query.OrderBy == "" {
		query.OrderBy = "name"
	}
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.Page == 0 {
		query.Page = 1
	}
	sess.Asc(query.OrderBy).Limit(query.Limit, (query.Page-1)*query.Limit)

	sess.Join("LEFT", "check", "endpoint.id = `check`.endpoint_id")
	sess.Join("LEFT", "endpoint_tag", "endpoint.id = endpoint_tag.endpoint_id")

	err := sess.Find(&e)
	if err != nil {
		return nil, err
	}
	return e.ToDTO(), nil
}

func GetEndpointById(id int64, orgId int64) (*model.EndpointDTO, error) {
	sess, err := newSession(false, "endpoint")
	if err != nil {
		return nil, err
	}
	return getEndpointById(sess, id, orgId)
}

func getEndpointById(sess *session, id int64, orgId int64) (*model.EndpointDTO, error) {
	var e endpointRows
	sess.Where("endpoint.id=? AND endpoint.org_id=?", id, orgId)
	sess.Join("LEFT", "check", "endpoint.id = `check`.endpoint_id")
	sess.Join("LEFT", "endpoint_tag", "endpoint.id = endpoint_tag.endpoint_id")

	err := sess.Find(&e)
	if err != nil {
		return nil, err
	}
	if len(e) == 0 {
		return nil, nil
	}
	return e.ToDTO()[0], nil
}

func AddEndpoint(e *model.EndpointDTO) error {
	sess, err := newSession(true, "endpoint")
	if err != nil {
		return err
	}
	defer sess.Cleanup()

	if err = addEndpoint(sess, e); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func addEndpoint(sess *session, e *model.EndpointDTO) error {
	endpoint := &model.Endpoint{
		OrgId:   e.OrgId,
		Name:    e.Name,
		Created: time.Now(),
		Updated: time.Now(),
	}
	endpoint.UpdateSlug()
	if _, err := sess.Insert(endpoint); err != nil {
		return err
	}
	e.Id = endpoint.Id
	e.Created = endpoint.Created
	e.Updated = endpoint.Updated
	e.Slug = endpoint.Slug

	endpointTags := make([]model.EndpointTag, 0, len(e.Tags))
	for _, tag := range e.Tags {
		endpointTags = append(endpointTags, model.EndpointTag{
			OrgId:      e.OrgId,
			EndpointId: endpoint.Id,
			Tag:        tag,
			Created:    time.Now(),
		})
	}
	if len(endpointTags) > 0 {
		sess.Table("endpoint_tag")
		if _, err := sess.Insert(&endpointTags); err != nil {
			return err
		}
	}

	//perform each insert on its own so that the ID field gets assigned and task created
	for _, c := range e.Checks {
		c.OrgId = e.OrgId
		c.EndpointId = e.Id
		if err := addCheck(sess, c); err != nil {
			return err
		}
	}

	return nil
}

func UpdateEndpoint(e *model.EndpointDTO) error {
	sess, err := newSession(true, "endpoint")
	if err != nil {
		return err
	}
	defer sess.Cleanup()

	if err = updateEndpoint(sess, e); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func updateEndpoint(sess *session, e *model.EndpointDTO) error {
	existing, err := getEndpointById(sess, e.Id, e.OrgId)
	if err != nil {
		return err
	}
	if existing == nil {
		return model.ErrEndpointNotFound
	}
	endpoint := &model.Endpoint{
		Id:      e.Id,
		OrgId:   e.OrgId,
		Name:    e.Name,
		Created: existing.Created,
		Updated: time.Now(),
	}
	endpoint.UpdateSlug()
	if _, err := sess.Id(endpoint.Id).Update(endpoint); err != nil {
		return err
	}

	e.Slug = endpoint.Slug
	e.Updated = endpoint.Updated

	/***** Update Tags **********/

	tagMap := make(map[string]bool)
	tagsToDelete := make([]string, 0)
	tagsToAddMap := make(map[string]bool, 0)
	// create map of current tags
	for _, t := range existing.Tags {
		tagMap[t] = false
	}

	// create map of tags to add. We use a map
	// to ensure that we only add each tag once.
	for _, t := range e.Tags {
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
		sess.Table("endpoint_tag")
		sess.Where("endpoint_id=? AND org_id=?", e.Id, e.OrgId)
		sess.In("tag", tagsToDelete)
		if _, err := sess.Delete(nil); err != nil {
			return err
		}
	}
	if len(tagsToAdd) > 0 {
		newEndpointTags := make([]model.EndpointTag, len(tagsToAdd))
		for i, tag := range tagsToAdd {
			newEndpointTags[i] = model.EndpointTag{
				OrgId:      e.OrgId,
				EndpointId: e.Id,
				Tag:        tag,
				Created:    time.Now(),
			}
		}
		sess.Table("endpoint_tag")
		if _, err := sess.Insert(&newEndpointTags); err != nil {
			return err
		}
	}

	/***** Update Checks **********/

	checkUpdates := make([]*model.Check, 0)
	checkAdds := make([]*model.Check, 0)
	checkDeletes := make([]*model.Check, 0)

	checkMap := make(map[model.CheckType]*model.Check)
	seenChecks := make(map[model.CheckType]bool)
	for _, c := range existing.Checks {
		checkMap[c.Type] = c
	}
	for _, c := range e.Checks {
		c.EndpointId = e.Id
		c.OrgId = e.OrgId
		seenChecks[c.Type] = true
		ec, ok := checkMap[c.Type]
		if !ok {
			checkAdds = append(checkAdds, c)
		} else if c.Id == ec.Id {
			cjson, err := json.Marshal(c)
			if err != nil {
				return err
			}
			ecjson, err := json.Marshal(ec)
			if !bytes.Equal(ecjson, cjson) {
				c.Created = ec.Created
				c.TaskId = ec.TaskId
				checkUpdates = append(checkAdds, c)
			}
		} else {
			return fmt.Errorf("Invalid check definition.")
		}
	}
	for t, ec := range checkMap {
		if _, ok := seenChecks[t]; !ok {
			checkDeletes = append(checkDeletes, ec)
		}
	}

	for _, c := range checkDeletes {
		if err := deleteCheck(sess, c); err != nil {
			return err
		}
	}

	for _, c := range checkAdds {
		if err := addCheck(sess, c); err != nil {
			return err
		}
	}

	for _, c := range checkUpdates {
		if err := updateCheck(sess, c); err != nil {
			return err
		}
	}

	return nil
}

func DeleteEndpoint(id, orgId int64) error {
	sess, err := newSession(true, "endpoint")
	if err != nil {
		return err
	}
	defer sess.Cleanup()

	if err = deleteEndpoint(sess, id, orgId); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func deleteEndpoint(sess *session, id, orgId int64) error {
	existing, err := getEndpointById(sess, id, orgId)
	if err != nil {
		return err
	}
	if existing == nil {
		return model.ErrEndpointNotFound
	}
	var rawSql = "DELETE FROM endpoint WHERE id=? and org_id=?"
	_, err = sess.Exec(rawSql, id, orgId)
	if err != nil {
		return err
	}

	rawSql = "DELETE FROM endpoint_tag WHERE endpoint_id=? and org_id=?"
	if _, err := sess.Exec(rawSql, id, orgId); err != nil {
		return err
	}
	checks := make([]*model.Check, 0)
	sess.Table("check")
	sess.Where("endpoint_id=?", id)
	if err := sess.Find(&checks); err != nil {
		return err
	}

	for _, c := range checks {
		if err := deleteCheck(sess, c); err != nil {
			return err
		}
	}
	return nil
}

func addCheck(sess *session, c *model.Check) error {
	c.State = -1
	c.StateCheck = time.Now()
	c.Created = time.Now()
	c.Updated = time.Now()
	sess.Table("check")
	sess.UseBool("enabled")
	_, err := sess.Insert(c)
	return err
}

func updateCheck(sess *session, c *model.Check) error {
	c.Updated = time.Now()
	sess.Table("check")
	sess.UseBool("enabled")
	_, err := sess.Id(c.Id).Update(c)
	return err
}

func deleteCheck(sess *session, c *model.Check) error {
	sess.Table("check")
	_, err := sess.Delete(c)
	return err
}
