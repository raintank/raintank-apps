package sqlstore

import (
	"time"

	"github.com/raintank/raintank-apps/server/model"
)

func GetMetricById(id string) (*model.Metric, error) {
	sess, err := newSession(false, "metric")
	if err != nil {
		return nil, err
	}

	return getMetricById(sess, id)
}

func getMetricById(sess *session, id string) (*model.Metric, error) {
	m := &model.Metric{}
	exists, err := sess.Id(id).Get(m)
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
	existing, err := getMetricById(sess, m.Id)
	if err != nil {
		return err
	}
	// no existing metric, creeate a new one.
	if existing != nil {
		m.Created = existing.Created
		return nil
	}
	m.Created = time.Now()
	if _, err := sess.Insert(m); err != nil {
		return err
	}
	return nil
}
