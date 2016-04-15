package main

import (
	"encoding/json"
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
	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/pkg/session"
	"github.com/raintank/raintank-apps/task-agent/snap"
	"github.com/rakyll/globalconf"
)

const Version int = 1

var (
	GitHash     = "(none)"
	showVersion = flag.Bool("version", false, "print version string")
	logLevel    = flag.Int("log-level", 2, "log level. 0=TRACE|1=DEBUG|2=INFO|3=WARN|4=ERROR|5=CRITICAL|6=FATAL")
	confFile    = flag.String("config", "/etc/raintank/collector.ini", "configuration file path")

	serverAddr = flag.String("server-url", "ws://localhost:80/api/v1/", "addres of raintank-apps server")
	tsdbAddr   = flag.String("tsdb-url", "http://localhost:80/", "addres of raintank-apps server")
	snapUrlStr = flag.String("snap-url", "http://localhost:8181", "url of SNAP server.")
	nodeName   = flag.String("name", "", "agent-name")
	apiKey     = flag.String("api-key", "not_very_secret_key", "Api Key")
)

func connect(u *url.URL) (*websocket.Conn, error) {
	log.Info("connecting to %s", u.String())
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("Bearer %s", *apiKey))
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	return conn, err
}

func main() {
	flag.Parse()
	// Only try and parse the conf file if it exists
	if _, err := os.Stat(*confFile); err == nil {
		conf, err := globalconf.NewWithOptions(&globalconf.Options{Filename: *confFile})
		if err != nil {
			panic(fmt.Sprintf("error with configuration file: %s", err))
		}
		conf.ParseAll()
	}

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

	if *showVersion {
		fmt.Printf("task-agent (built with %s, git hash %s)\n", runtime.Version(), GitHash)
		return
	}

	if *nodeName == "" {
		log.Fatal(4, "name must be set.")
	}

	snapUrl, err := url.Parse(*snapUrlStr)
	if err != nil {
		log.Fatal(4, "could not parse snapUrl. %s", err)
	}
	snapClient, err := snap.NewClient(*nodeName, *tsdbAddr, *apiKey, snapUrl)
	if err != nil {
		log.Fatal(4, err.Error())
	}
	InitTaskCache(snapClient)

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
		log.Debug("recieved heartbeat event. %s", body)
	})

	sess.On("taskUpdate", HandleTaskUpdate())
	sess.On("taskAdd", HandleTaskAdd())
	sess.On("taskRemove", HandleTaskRemove())

	go sess.Start()

	//periodically send an Updated Catalog.
	go SendCatalog(sess, snapClient, shutdownStart)

	// connect to the snap server and monitor that it is up.
	go snapClient.Run()

	//wait for interupt Signal.
	<-interrupt
	log.Info("interrupt")
	close(shutdownStart)
	sess.Close()
	return
}

func SendCatalog(sess *session.Session, snapClient *snap.Client, shutdownStart chan struct{}) {
	ticker := time.NewTicker(time.Minute * 5)
	for {
		select {
		case <-shutdownStart:
			return
		case <-ticker.C:
			emitMetrics(sess, snapClient)
		case <-snapClient.ConnectChan:
			emitMetrics(sess, snapClient)
			taskList, err := snapClient.GetSnapTasks()
			if err != nil {
				log.Error(3, err.Error())
				continue
			}
			if err := GlobalTaskCache.IndexSnapTasks(taskList); err != nil {
				log.Error(3, "failed to add task to cache. %s", err)
			}
		}
	}
}

func emitMetrics(sess *session.Session, snapClient *snap.Client) {
	catalog, err := snapClient.GetSnapMetrics()
	if err != nil {
		log.Error(3, err.Error())
		return
	}
	body, err := json.Marshal(catalog)
	if err != nil {
		log.Error(3, err.Error())
		return
	}
	e := &message.Event{Event: "catalog", Payload: body}
	sess.Emit(e)
}
