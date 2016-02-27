package sqlstore

import (
	"time"

	"github.com/raintank/raintank-apps/server/model"
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

func GetMetricById(id string, owner string) (*model.Metric, error) {
	sess, err := newSession(false, "metric")
	if err != nil {
		return nil, err
	}

	return getMetricById(sess, id, owner)
}

func getMetricById(sess *session, id string, owner string) (*model.Metric, error) {
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
	m.SetId()
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
	m.Created = time.Now()
	if _, err := sess.Insert(m); err != nil {
		return err
	}
	return nil
}
