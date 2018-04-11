package client

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/codeskyblue/go-uuid"
	"github.com/raintank/raintank-apps/task-server/api"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
	"github.com/raintank/worldping-api/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	adminKey = "changeme"
	lock     sync.RWMutex
)

func startApi(done chan struct{}) string {
	log.NewLogger(0, "console", fmt.Sprintf(`{"level": %d, "formatting":true}`, 1))

	sqlstore.NewEngine("sqlite3", ":memory:", true)
	//sqlstore.NewEngine("sqlite3", "file:/tmp/task-server-test.db?cache=shared&mode=rwc&_loc=Local", true)

	addTestData()

	m := api.NewApi(adminKey)

	// define our own listner so we can call Close on it
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err.Error())
	}

	go http.Serve(l, m)
	go func() {
		<-done
		l.Close()
	}()

	return fmt.Sprintf("http://%s/", l.Addr().String())
}

func addTestData() {
	// add public agent

	agent := &model.AgentDTO{
		Name:    "publicTest",
		Enabled: true,
		OrgId:   1000,
		Public:  true,
	}
	err := sqlstore.AddAgent(agent)
	if err != nil {
		panic(err.Error())
	}
	err = sqlstore.AddAgentSession(&model.AgentSession{
		Id:       uuid.NewUUID().String(),
		AgentId:  agent.Id,
		Version:  1,
		RemoteIp: "127.0.0.1",
		Server:   "localhost",
		Created:  time.Now(),
	})
	if err != nil {
		panic(err)
	}
}

func addTestMetrics(agent *model.AgentDTO) {
	err := sqlstore.AddAgentSession(&model.AgentSession{
		Id:       uuid.NewUUID().String(),
		AgentId:  agent.Id,
		Version:  1,
		RemoteIp: "127.0.0.1",
		Server:   "localhost",
		Created:  time.Now(),
	})
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
	agentCount := 1
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
				Public:  false,
				Tags:    []string{"demo", "private"},
			}

			aErr := c.AddAgent(&a)

			So(aErr, ShouldBeNil)
			So(a.Id, ShouldNotBeEmpty)
			So(a.Name, ShouldEqual, fmt.Sprintf("demo%d", agentCount))
			So(a.Enabled, ShouldEqual, true)
			So(a.Public, ShouldEqual, false)
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
					So(a.Public, ShouldEqual, false)
					So(a.Created, ShouldHappenBefore, pre)
					So(a.Updated, ShouldHappenAfter, pre)
				})
				Convey("When deleting an agent", func() {
					err := c.DeleteAgent(&a)
					So(err, ShouldBeNil)
					agentCount--

					Convey("When searching for agent by name", func() {
						query := model.GetAgentsQuery{Name: a.Name}
						agents, err := c.GetAgents(&query)
						So(err, ShouldBeNil)
						So(len(agents), ShouldEqual, 0)
					})
				})
			})
			Convey("When getting the list of Agents", func() {
				query := model.GetAgentsQuery{}
				agents, err := c.GetAgents(&query)

				So(err, ShouldBeNil)
				So(len(agents), ShouldEqual, agentCount)
			})
		})

		Convey("When getting list of public agents", func() {

			query := model.GetAgentsQuery{Public: "true"}
			agents, err := c.GetAgents(&query)

			So(err, ShouldBeNil)
			So(len(agents), ShouldEqual, 1)
			So(agents[0].Id, ShouldEqual, 1)

			Convey("When updating tags of public agent", func() {
				a := new(model.AgentDTO)
				*a = *agents[0]
				a.Tags = []string{"foo", "demo"}
				err := c.UpdateAgent(a)
				So(err, ShouldBeNil)
				So(a.Id, ShouldNotBeEmpty)
				So(a.Name, ShouldEqual, "publicTest")
				So(a.Enabled, ShouldEqual, true)
				So(a.Public, ShouldEqual, true)
				So(len(a.Tags), ShouldEqual, 2)
			})
		})

		Convey("When getting list of tasks", func() {
			query := model.GetTasksQuery{}
			tasks, err := c.GetTasks(&query)
			So(err, ShouldBeNil)
			//So(tasks, ShouldNotBeNil)
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
			/* Skip this for now
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
			*/
			Convey("When Adding new Task with route by tag", func() {
				t := &model.TaskDTO{
					Name:     "task route by tags",
					Interval: 60,
					Config: map[string]map[string]interface{}{"/": {
						"user":   "test",
						"passwd": "test",
					}},
					Route: &model.TaskRoute{
						Type:   model.RouteByTags,
						Config: map[string]interface{}{"tags": []string{"demo"}},
					},
					Enabled: true,
				}
				taskCount++
				err = c.AddTask(t)
				So(err, ShouldBeNil)

				Convey("When getting agentTasks", func() {
					tasks, err := sqlstore.GetAgentTasks(&model.AgentDTO{Id: 1, OrgId: 1000})
					So(err, ShouldBeNil)
					So(len(tasks), ShouldEqual, 3)
					So(tasks[2].Name, ShouldEqual, "task route by tags")
					tasks, err = sqlstore.GetAgentTasks(&model.AgentDTO{Id: 2, OrgId: 1})
					So(err, ShouldBeNil)
					So(len(tasks), ShouldEqual, 1)
				})
			})
			Convey("When Adding new Task with route by tag matching only private probes", func() {
				t := &model.TaskDTO{
					Name:     "task route by tags2",
					Interval: 60,
					Config: map[string]map[string]interface{}{"/": {
						"user":   "test",
						"passwd": "test",
					}},
					Route: &model.TaskRoute{
						Type:   model.RouteByTags,
						Config: map[string]interface{}{"tags": []string{"private"}},
					},
					Enabled: true,
				}
				taskCount++
				err = c.AddTask(t)
				So(err, ShouldBeNil)

				Convey("When getting agentTasks", func() {
					tasks, err := sqlstore.GetAgentTasks(&model.AgentDTO{Id: 1, OrgId: 1000})
					So(err, ShouldBeNil)
					So(len(tasks), ShouldEqual, 3)
					So(tasks[2].Name, ShouldEqual, "task route by tags")
					tasks, err = sqlstore.GetAgentTasks(&model.AgentDTO{Id: 2, OrgId: 1})
					So(err, ShouldBeNil)
					So(len(tasks), ShouldEqual, 2)
				})
			})
			Convey("When Adding new Task with no valid agents", func() {
				err := sqlstore.DeleteAgentSessionsByServer("localhost")
				So(err, ShouldBeNil)

				t := &model.TaskDTO{
					Name:     "task should fail",
					Interval: 60,
					Config: map[string]map[string]interface{}{"/": {
						"user":   "test",
						"passwd": "test",
					}},
					Route: &model.TaskRoute{
						Type: "any",
					},
					Enabled: true,
				}
				err = c.AddTask(t)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "500: No agent found that can provide all requested metrics.")
			})
		})
	})
}
