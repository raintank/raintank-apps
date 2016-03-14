package sqlstore

import (
	"time"

	"github.com/raintank/raintank-apps/task-server/model"
)

func GetMetrics(query *model.GetMetricsQuery) ([]*model.Metric, error) {
	sess, err := newSession(false, "metric")
	if err != nil {
		return nil, err
	}
	return getMetrics(sess, query)
}

func getMetrics(sess *session, query *model.GetMetricsQuery) ([]*model.Metric, error) {
	metrics := make([]*model.Metric, 0)
	sess.Where("(public=1 OR owner = ?)", query.Owner)
	if query.Namespace != "" {
		sess.And("namespace like ?", query.Namespace)
	}
	if query.Version != 0 {
		sess.And("version = ?", query.Version)
	}
	err := sess.Find(&metrics)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func GetMetricById(id string, owner int64) (*model.Metric, error) {
	sess, err := newSession(false, "metric")
	if err != nil {
		return nil, err
	}

	return getMetricById(sess, id, owner)
}

func getMetricById(sess *session, id string, owner int64) (*model.Metric, error) {
	m := &model.Metric{}
	exists, err := sess.Where("(public=1 OR owner = ?) AND id=?", owner, id).Get(m)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return m, nil
}

func AddMetric(m *model.Metric) error {
	sess, err := newSession(true, "metric")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = addMetric(sess, m); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func addMetric(sess *session, m *model.Metric) error {
	m.SetId()
	existing := &model.Metric{}
	exists, err := sess.Id(m.Id).Get(existing)
	if err != nil {
		return err
	}
	if exists {
		return model.MetricAlreadyExists
	}
	m.Created = time.Now()
	if _, err := sess.Insert(m); err != nil {
		return err
	}
	return nil
}

func AddMissingMetrics(m []*model.Metric) error {
	sess, err := newSession(true, "metric")
	if err != nil {
		return err
	}
	defer sess.Cleanup()
	if err = addMissingMetrics(sess, m); err != nil {
		return err
	}
	sess.Complete()
	return nil
}

func addMissingMetrics(sess *session, metrics []*model.Metric) error {
	existing := make([]*model.Metric, 0)
	ids := make([]string, len(metrics))
	for i, m := range metrics {
		m.SetId()
		ids[i] = m.Id
	}
	sess.In("id", ids)
	err := sess.Find(&existing)
	if err != nil {
		return err
	}
	existingMap := make(map[string]struct{})
	for _, m := range existing {
		existingMap[m.Id] = struct{}{}
	}

	toAdd := make([]*model.Metric, 0)
	for _, m := range metrics {
		if _, ok := existingMap[m.Id]; !ok {
			m.Created = time.Now()
			toAdd = append(toAdd, m)
		}
	}
	if len(toAdd) > 0 {
		if _, err := sess.Insert(&metrics); err != nil {
			return err
		}
	}
	return nil
}
