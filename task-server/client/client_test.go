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

func addTestMetrics(agent *model.AgentDTO) {
	metrics := []*model.Metric{
		{
			OrgId:     1,
			Public:    true,
			Namespace: "/testing/demo/demo1",
			Version:   1,
			Policy: []rbody.PolicyTable{
				{
					Name:     "user",
					Type:     "string",
					Required: true,
				},
				{
					Name:     "passwd",
					Type:     "string",
					Required: true,
				},
				{
					Name:     "limit",
					Type:     "integer",
					Required: false,
					Default:  10,
				},
			},
		},
		{
			OrgId:     1,
			Public:    true,
			Namespace: "/testing/demo2/demo",
			Version:   2,
			Policy:    nil,
		},
	}
	err := sqlstore.AddMissingMetricsForAgent(agent, metrics)
	if err != nil {
		panic(err)
	}
}

func TestApiClient(t *testing.T) {
	done := make(chan struct{})
	defer func() {
		close(done)
		time.Sleep(time.Second)
	}()
	url := startApi(done)
	agentCount := 0
	metricsCount := 0
	taskCount := 0
	Convey("Client should exist", t, func() {
		c, cerr := New(url, adminKey, false)
		So(cerr, ShouldBeNil)
		Convey("When calling the api heartbeat method", func() {
			ok, hErr := c.Heartbeat()
			So(hErr, ShouldBeNil)
			So(ok, ShouldBeTrue)
		})

		Convey("when adding a new Agent", func() {
			agentCount++
			pre := time.Now()
			a := model.AgentDTO{
				Name:    fmt.Sprintf("demo%d", agentCount),
				Enabled: true,
				Public:  true,
				Tags:    []string{"demo", "test"},
			}

			aErr := c.AddAgent(&a)

			So(aErr, ShouldBeNil)
			So(a.Id, ShouldNotBeEmpty)
			So(a.Name, ShouldEqual, fmt.Sprintf("demo%d", agentCount))
			So(a.Enabled, ShouldEqual, true)
			So(a.Public, ShouldEqual, true)
			So(a.Created, ShouldHappenBefore, time.Now())
			So(a.Created, ShouldHappenAfter, pre)
			So(a.Created.Unix(), ShouldEqual, a.Updated.Unix())

			Convey("when getting an agent by id", func() {
				agent, err := c.GetAgentById(a.Id)
				So(err, ShouldBeNil)
				So(agent, ShouldNotBeNil)
				So(agent, ShouldHaveSameTypeAs, &model.AgentDTO{})
				So(agent.Id, ShouldEqual, a.Id)
				So(agent.Created.Unix(), ShouldEqual, a.Created.Unix())
				Convey("when updating an Agent", func() {
					a := new(model.AgentDTO)
					*a = *agent
					a.Name = "test1"
					pre := time.Now()
					err := c.UpdateAgent(a)
					So(err, ShouldBeNil)
					So(a.Id, ShouldNotBeEmpty)
					So(a.Name, ShouldEqual, "test1")
					So(a.Enabled, ShouldEqual, true)
					So(a.Public, ShouldEqual, true)
					So(a.Created, ShouldHappenBefore, pre)
					So(a.Updated, ShouldHappenAfter, pre)
				})
			})
			var deleteTime time.Time
			Convey("When getting the list of Agents", func() {
				query := model.GetAgentsQuery{}
				agents, err := c.GetAgents(&query)

				So(err, ShouldBeNil)
				So(len(agents), ShouldEqual, agentCount)
				So(agents[0].Name, ShouldEqual, "demo2")

				Convey("When deleting an agent", func() {
					err := c.DeleteAgent(agents[0])
					So(err, ShouldBeNil)
					agentCount--
					deleteTime = time.Now()
				})
				Convey("After deleting agent", func() {
					//agent demo2 was deleted, then re-added when
					// "when adding a new Agent" was run prior to this block.
					So(agents[0].Created, ShouldHappenAfter, deleteTime)
				})
			})
		})

		// Metric Tests
		Convey("When getting metrics list", func() {
			query := &model.GetMetricsQuery{}
			metrics, err := c.GetMetrics(query)
			So(err, ShouldBeNil)
			So(metrics, ShouldNotBeNil)
			So(metrics, ShouldHaveSameTypeAs, []*model.Metric{})
			So(len(metrics), ShouldEqual, metricsCount)
			agents, err := c.GetAgents(&model.GetAgentsQuery{})
			if err != nil {
				panic(err)
			}
			addTestMetrics(agents[0])
			metricsCount = 2
			Convey("When getting metrics for Agent", func() {
				metrics, err := c.GetAgentMetrics(agents[0].Id)
				So(err, ShouldBeNil)
				So(metrics, ShouldNotBeNil)
				So(metrics, ShouldHaveSameTypeAs, []*model.Metric{})
				So(len(metrics), ShouldEqual, 2)
			})
			Convey("When getting agent with Metric", func() {
				q := &model.GetAgentsQuery{
					Metric: "/testing/demo/demo1",
				}
				agentsWithMetric, err := c.GetAgents(q)
				So(err, ShouldBeNil)
				So(agentsWithMetric, ShouldNotBeNil)
				So(agentsWithMetric, ShouldHaveSameTypeAs, []*model.AgentDTO{})
				So(len(agentsWithMetric), ShouldEqual, 1)
				So(agentsWithMetric[0].Id, ShouldEqual, agents[0].Id)
			})
		})

		Convey("When getting list of tasks", func() {
			query := model.GetTasksQuery{}
			tasks, err := c.GetTasks(&query)
			So(err, ShouldBeNil)
			So(tasks, ShouldNotBeNil)
			So(len(tasks), ShouldEqual, taskCount)
			So(tasks, ShouldHaveSameTypeAs, []*model.TaskDTO{})
			Convey("When Adding new Task", func() {
				pre := time.Now()
				taskCount++
				t := &model.TaskDTO{
					Name:     fmt.Sprintf("test Task%d", taskCount),
					Interval: 60,
					Config: map[string]map[string]interface{}{"/": {
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
				So(t.Name, ShouldEqual, fmt.Sprintf("test Task%d", taskCount))
				So(t.Created, ShouldHappenBefore, time.Now())
				So(t.Created, ShouldHappenAfter, pre)
				So(t.Created.Unix(), ShouldEqual, t.Updated.Unix())
				Convey("When adding first task", func() {
					So(len(tasks), ShouldEqual, 0)
				})
				Convey("When adding second task", func() {
					So(len(tasks), ShouldEqual, 1)
				})

			})
			Convey("when updating task", func() {
				pre := time.Now()
				t := new(model.TaskDTO)
				*t = *tasks[0]
				t.Name = "demo"
				err := c.UpdateTask(t)
				So(err, ShouldBeNil)
				So(t.Id, ShouldEqual, tasks[0].Id)
				So(t.Name, ShouldEqual, "demo")
				So(t.Created, ShouldHappenBefore, pre)
				So(t.Updated, ShouldHappenAfter, pre)
				So(t.Updated, ShouldHappenAfter, t.Created)
			})
		})
	})
}
