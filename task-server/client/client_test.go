package client

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

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

func TestApiClient(t *testing.T) {
	done := make(chan struct{})
	url := startApi(done)
	c, cerr := New(url, adminKey, false)
	Convey("Client should exist", t, func() {
		So(cerr, ShouldBeNil)
		Convey("Testing API after startup", func() {
			ok, hErr := c.Heartbeat()
			So(hErr, ShouldBeNil)
			So(ok, ShouldBeTrue)
		})
		Convey("Adding new Agent", func() {
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
		Convey("Getting Agents", func() {
			query := model.GetAgentsQuery{}
			agents, err := c.GetAgents(&query)
			So(err, ShouldBeNil)
			So(len(agents), ShouldEqual, 1)
			So(agents[0].Name, ShouldEqual, "demo1")
			Convey("Getting AgentById", func() {
				agent, err := c.GetAgentById(agents[0].Id)
				So(err, ShouldBeNil)
				So(agent, ShouldNotBeNil)
				So(agent, ShouldHaveSameTypeAs, &model.AgentDTO{})
				So(agent.Id, ShouldEqual, agents[0].Id)
				So(agent.Created.Unix(), ShouldEqual, agents[0].Created.Unix())
			})
			Convey("Updating Agent", func() {
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

	})
	close(done)
}
