package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
	"github.com/raintank/raintank-apps/pkg/session"
	taConfig "github.com/raintank/raintank-apps/task-agent-ng/taskagentconfig"

	"github.com/raintank/worldping-api/pkg/log"
	"github.com/rakyll/globalconf"
)

const Version int = 1

var (
	GitHash           = "(none)"
	showVersion       = flag.Bool("version", false, "print version string")
	logLevel          = flag.Int("log-level", 2, "log level. 0=TRACE|1=DEBUG|2=INFO|3=WARN|4=ERROR|5=CRITICAL|6=FATAL")
	confFile          = flag.String("config", "/etc/raintank/collector.ini", "configuration file path")
	serverAddr        = flag.String("server-url", "ws://localhost:8082/api/v1/", "address of raintank-apps server")
	tsdbgwAddr        = flag.String("tsdbgw-url", "http://localhost:8082/", "address of a tsdb-gw server")
	tsdbgwAdminAPIKey = flag.String("tsdbgw-admin-key", "tsdbgw_not_very_secret_key", "admin key used to post to tsdb-gw")
	nodeName          = flag.String("name", "", "agent-name")
	appAPIKey         = flag.String("app-api-key", "app_not_very_secret_key", "API Key for task-server and task-agent communication")
)

func connect(u *url.URL) (*websocket.Conn, error) {
	log.Info("connecting to %s", u.String())
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("Bearer %s", *appAPIKey))
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	return conn, err
}

func main() {
	flag.Parse()
	// Set 'cfile' here if *confFile exists, because we should only try and
	// parse the conf file if it exists. If we try and parse the default
	// conf file location when it's not there, we (unsurprisingly) get a
	// panic.
	var cfile string
	if _, err := os.Stat(*confFile); err == nil {
		cfile = *confFile
	}

	// Still parse globalconf, though, even if the config file doesn't exist
	// because we want to be able to use environment variables.
	config, err := globalconf.NewWithOptions(&globalconf.Options{
		Filename:  cfile,
		EnvPrefix: "TASKAGENT_",
	})
	if err != nil {
		panic(fmt.Sprintf("error with configuration file: %s", err))
	}
	if *showVersion {
		fmt.Printf("task-agent-ng (built with %s, git hash %s)\n", runtime.Version(), GitHash)
		return
	}
	taConfig.ConfigSetup()
	config.ParseAll()

	log.NewLogger(0, "console", fmt.Sprintf(`{"level": %d, "formatting":true}`, *logLevel))
	// workaround for https://github.com/grafana/grafana/issues/4055
	switch *logLevel {
	case 0:
		log.Level(log.TRACE)
	case 1:
		log.Level(log.DEBUG)
	case 2:
		log.Level(log.INFO)
	case 3:
		log.Level(log.WARN)
	case 4:
		log.Level(log.ERROR)
	case 5:
		log.Level(log.CRITICAL)
	case 6:
		log.Level(log.FATAL)
	}

	//*nodeName = "agent1"
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(4, "failed to get hostname from OS.")
	}
	taConfig.ConfigProcess(hostname)
	taConfig.Start()

	if hostname == "" {
		log.Fatal(4, "name must be set.")
	}

	InitTaskCache(tsdbgwAddr, tsdbgwAdminAPIKey)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	shutdownStart := make(chan struct{})

	controllerUrl, err := url.Parse(*serverAddr)
	if err != nil {
		log.Fatal(4, err.Error())
	}
	controllerUrl.Path = path.Clean(controllerUrl.Path + fmt.Sprintf("/socket/%s/%d", *nodeName, Version))

	if controllerUrl.Scheme != "ws" && controllerUrl.Scheme != "wss" {
		log.Fatal(4, "invalid server address.  scheme must be ws or wss. was %s", controllerUrl.Scheme)
	}

	conn, err := connect(controllerUrl)
	if err != nil {
		log.Fatal(4, "unable to connect to server on url %s: %s", controllerUrl.String(), err)
	}

	//create new session, allow 1000 events to be queued in the writeQueue before Emit() blocks.
	sess := session.NewSession(conn, 1000)
	sess.On("disconnect", func() {
		// on disconnect, reconnect.
		ticker := time.NewTicker(time.Second)
		connected := false
		for !connected {
			select {
			case <-shutdownStart:
				ticker.Stop()
				return
			case <-ticker.C:
				conn, err := connect(controllerUrl)
				if err == nil {
					sess.Conn = conn
					connected = true
					go sess.Start()
				}
			}
		}
		ticker.Stop()
	})

	sess.On("heartbeat", func(body []byte) {
		log.Debug("received heartbeat event. %s", body)
	})

	sess.On("taskList", HandleTaskList())
	sess.On("taskUpdate", HandleTaskUpdate())
	sess.On("taskAdd", HandleTaskAdd())
	sess.On("taskRemove", HandleTaskRemove())

	go sess.Start()

	//wait for interrupt Signal.
	<-interrupt
	log.Info("interrupt")
	close(shutdownStart)
	sess.Close()
	return
}
