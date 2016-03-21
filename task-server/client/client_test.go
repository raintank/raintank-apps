package client

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/intelsdi-x/snap/mgmt/rest/rbody"
	"github.com/raintank/met/helper"
	"github.com/raintank/raintank-apps/task-server/api"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	adminKey = "changeme"
)

func startApi(done chan struct{}) string {
	stats, err := helper.New(false, "localhost:8125", "standard", "task-server", "default")
	if err != nil {
		panic(fmt.Errorf("failed to initialize statsd. %s", err))
	}

	// initialize DB
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		panic(err.Error)
	}
	dbpath := tmpfile.Name()
	tmpfile.Close()
	fmt.Printf("dbpath: %s\n", dbpath)
	sqlstore.NewEngine(dbpath)

	m := api.NewApi(adminKey, stats)

	// define our own listner so we can call Close on it
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err.Error())
	}

	go http.Serve(l, m)
	go func() {
		<-done
		l.Close()
		os.Remove(dbpath)
	}()

	return fmt.Sprintf("http://%s/", l.Addr().String())
}

func addTestMetrics() {
	metrics := []*model.Metric{
		&model.Metric{
			Owner:     1,
			Public:    true,
			Namespace: "/testing/demo/demo1",
			Version:   1,
			Policy: []rbody.PolicyTable{
				rbody.PolicyTable{
					Name:     "user",
					Type:     "string",
					Required: true,
				},
				rbody.PolicyTable{
					Name:     "passwd",
					Type:     "string",
					Required: true,
				},
				rbody.PolicyTable{
					Name:     "limit",
					Type:     "integer",
					Required: false,
					Default:  10,
				},
			},
		},
		&model.Metric{
			Owner:     1,
			Public:    true,
			Namespace: "/testing/demo2/demo",
			Version:   2,
			Policy:    nil,
		},
	}
	err := sqlstore.AddMissingMetrics(metrics)
	if err != nil {
		panic(err)
	}
}

func TestApiClient(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	url := startApi(done)
	c, cerr := New(url, adminKey, false)
	Convey("Client should exist", t, func() {
		So(cerr, ShouldBeNil)
		Convey("When calling the api heartbeat method", func() {
			ok, hErr := c.Heartbeat()
			So(hErr, ShouldBeNil)
			So(ok, ShouldBeTrue)
		})
		Convey("when adding a new Agent", func() {
			pre := time.Now()
			a := model.AgentDTO{
				Name:    "demo1",
				Enabled: true,
				Public:  true,
				Tags:    []string{"demo", "test"},
			}
			aErr := c.AddAgent(&a)
			So(aErr, ShouldBeNil)
			So(a.Id, ShouldNotBeEmpty)
			So(a.Name, ShouldEqual, "demo1")
			So(a.Enabled, ShouldEqual, true)
			So(a.Public, ShouldEqual, true)
			So(a.Created, ShouldHappenBefore, time.Now())
			So(a.Created, ShouldHappenAfter, pre)
			So(a.Created.Unix(), ShouldEqual, a.Updated.Unix())

		})
		Convey("When getting the list of Agents", func() {
			query := model.GetAgentsQuery{}
			agents, err := c.GetAgents(&query)

			So(err, ShouldBeNil)
			So(len(agents), ShouldEqual, 1)
			So(agents[0].Name, ShouldEqual, "demo1")
			Convey("when getting the 1 agent by id", func() {
				agent, err := c.GetAgentById(agents[0].Id)
				So(err, ShouldBeNil)
				So(agent, ShouldNotBeNil)
				So(agent, ShouldHaveSameTypeAs, &model.AgentDTO{})
				So(agent.Id, ShouldEqual, agents[0].Id)
				So(agent.Created.Unix(), ShouldEqual, agents[0].Created.Unix())
			})
			Convey("when updating the 1 Agent", func() {
				a := new(model.AgentDTO)
				*a = *agents[0]
				a.Name = "demo2"
				pre := time.Now()
				err := c.UpdateAgent(a)
				So(err, ShouldBeNil)
				So(a.Id, ShouldNotBeEmpty)
				So(a.Name, ShouldEqual, "demo2")
				So(a.Enabled, ShouldEqual, true)
				So(a.Public, ShouldEqual, true)
				So(a.Created, ShouldHappenBefore, pre)
				So(a.Updated, ShouldHappenAfter, pre)
			})
		})
		Convey("when adding a second Agent", func() {
			pre := time.Now()
			a := model.AgentDTO{
				Name:    "demo3",
				Enabled: true,
				Public:  true,
				Tags:    []string{"demo", "test"},
			}
			aErr := c.AddAgent(&a)
			So(aErr, ShouldBeNil)
			So(a.Id, ShouldNotBeEmpty)
			So(a.Name, ShouldEqual, "demo3")
			So(a.Enabled, ShouldEqual, true)
			So(a.Public, ShouldEqual, true)
			So(a.Created, ShouldHappenBefore, time.Now())
			So(a.Created, ShouldHappenAfter, pre)
			So(a.Created.Unix(), ShouldEqual, a.Updated.Unix())
		})
		Convey("When getting the list of 2 agents", func() {
			query := model.GetAgentsQuery{}
			agents, err := c.GetAgents(&query)
			So(err, ShouldBeNil)
			So(len(agents), ShouldEqual, 2)
			So(agents[0].Name, ShouldEqual, "demo2")
			Convey("When getting first Agent by id", func() {
				agent, err := c.GetAgentById(agents[0].Id)
				So(err, ShouldBeNil)
				So(agent, ShouldNotBeNil)
				So(agent, ShouldHaveSameTypeAs, &model.AgentDTO{})
				So(agent.Id, ShouldEqual, agents[0].Id)
				So(agent.Created.Unix(), ShouldEqual, agents[0].Created.Unix())
			})
			Convey("When deleting an agent", func() {
				err := c.DeleteAgent(agents[0])
				So(err, ShouldBeNil)
			})
		})
		Convey("Getting Agents list after delete", func() {
			query := model.GetAgentsQuery{}
			agents, err := c.GetAgents(&query)
			So(err, ShouldBeNil)
			So(len(agents), ShouldEqual, 1)
			So(agents[0].Name, ShouldEqual, "demo3")
		})

		Convey("When getting empty metrics list", func() {
			query := &model.GetMetricsQuery{}
			metrics, err := c.GetMetrics(query)
			So(err, ShouldBeNil)
			So(metrics, ShouldNotBeNil)
			So(metrics, ShouldHaveSameTypeAs, []*model.Metric{})
			So(len(metrics), ShouldEqual, 0)
		})
		Convey("When getting metrics list", func() {
			addTestMetrics()
			query := &model.GetMetricsQuery{}
			metrics, err := c.GetMetrics(query)
			So(err, ShouldBeNil)
			So(metrics, ShouldNotBeNil)
			So(metrics, ShouldHaveSameTypeAs, []*model.Metric{})
			So(len(metrics), ShouldEqual, 2)
		})

		Convey("When getting empty list of tasks", func() {
			query := model.GetTasksQuery{}
			tasks, err := c.GetTasks(&query)
			So(err, ShouldBeNil)
			So(tasks, ShouldNotBeNil)
			So(len(tasks), ShouldEqual, 0)
			So(tasks, ShouldHaveSameTypeAs, []*model.TaskDTO{})
		})
		Convey("When Adding new Task", func() {
			pre := time.Now()
			t := &model.TaskDTO{
				Name:     "test Task",
				Interval: 60,
				Config: map[string]map[string]interface{}{"/": map[string]interface{}{
					"user":   "test",
					"passwd": "test",
				}},
				Metrics: map[string]int64{"/testing/demo/demo1": 0},
				Route: &model.TaskRoute{
					Type: "any",
				},
				Enabled: true,
			}
			err := c.AddTask(t)
			So(err, ShouldBeNil)
			So(t.Id, ShouldNotBeEmpty)
			So(t.Name, ShouldEqual, "test Task")
			So(t.Created, ShouldHappenBefore, time.Now())
			So(t.Created, ShouldHappenAfter, pre)
			So(t.Created.Unix(), ShouldEqual, t.Updated.Unix())
		})
		Convey("When getting list of 1 task in db", func() {
			query := model.GetTasksQuery{}
			tasks, err := c.GetTasks(&query)
			So(err, ShouldBeNil)
			So(tasks, ShouldNotBeNil)
			So(len(tasks), ShouldEqual, 1)
			So(tasks, ShouldHaveSameTypeAs, []*model.TaskDTO{})
			So(tasks[0].Name, ShouldEqual, "test Task")
			Convey("when updating task", func() {
				pre := time.Now()
				t := new(model.TaskDTO)
				*t = *tasks[0]
				t.Name = "test Task2"
				err := c.UpdateTask(t)
				So(err, ShouldBeNil)
				So(t.Id, ShouldEqual, tasks[0].Id)
				So(t.Name, ShouldEqual, "test Task2")
				So(t.Created, ShouldHappenBefore, pre)
				So(t.Updated, ShouldHappenAfter, pre)
				So(t.Updated, ShouldHappenAfter, t.Created)
			})
		})
		Convey("When adding second task to DB", func() {
			pre := time.Now()
			t := &model.TaskDTO{
				Name:     "test Task3",
				Interval: 60,
				Config: map[string]map[string]interface{}{"/": map[string]interface{}{
					"user":   "test",
					"passwd": "test",
				}},
				Metrics: map[string]int64{"/testing/demo/demo1": 0},
				Route: &model.TaskRoute{
					Type: "any",
				},
				Enabled: true,
			}
			err := c.AddTask(t)
			So(err, ShouldBeNil)
			So(t.Id, ShouldNotBeEmpty)
			So(t.Name, ShouldEqual, "test Task3")
			So(t.Created, ShouldHappenBefore, time.Now())
			So(t.Created, ShouldHappenAfter, pre)
			So(t.Created.Unix(), ShouldEqual, t.Updated.Unix())

		})
		Convey("When getting list of 2 tasks in db", func() {
			query := model.GetTasksQuery{}
			tasks, err := c.GetTasks(&query)
			So(err, ShouldBeNil)
			So(tasks, ShouldNotBeNil)
			So(len(tasks), ShouldEqual, 2)
			So(tasks, ShouldHaveSameTypeAs, []*model.TaskDTO{})
			So(tasks[0].Name, ShouldEqual, "test Task2")
			Convey("when deleting a task", func() {
				err := c.DeleteTask(tasks[0])
				So(err, ShouldBeNil)
				Convey("When getting list of tasks after delete", func() {
					tasks, err = c.GetTasks(&query)
					So(err, ShouldBeNil)
					So(tasks, ShouldNotBeNil)
					So(len(tasks), ShouldEqual, 1)
					So(tasks, ShouldHaveSameTypeAs, []*model.TaskDTO{})
					So(tasks[0].Name, ShouldEqual, "test Task3")
				})
			})
		})
	})
}
