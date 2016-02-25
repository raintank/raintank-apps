package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/raintank-apps/pkg/session"
	"github.com/rakyll/globalconf"
)

const Version int = 1

var log = logging.MustGetLogger("default")

var (
	showVersion = flag.Bool("version", false, "print version string")
	logLevel    = flag.Int("log-level", 4, "log level. 5=DEBUG|4=INFO|3=NOTICE|2=WARNING|1=ERROR|0=CRITICAL")
	confFile    = flag.String("config", "/etc/raintank/collector.ini", "configuration file path")

	addr       = flag.String("addr", "localhost:8081", "http service address")
	snapUrlStr = flag.String("snap-url", "http://localhost:8181/", "url of SNAP server.")
	nodeName   = flag.String("name", "", "agent name")
)

func connect(u url.URL) (*websocket.Conn, error) {
	log.Infof("connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
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

	logging.SetFormatter(logging.GlogFormatter)
	logging.SetLevel(logging.Level(*logLevel), "default")
	log.SetBackend(logging.AddModuleLevel(logging.NewLogBackend(os.Stdout, "", 0)))

	if *nodeName == "" {
		log.Fatalf("name must be set.")
	}

	snapUrl, err := url.Parse(*snapUrlStr)
	if err != nil {
		log.Fatalf("could not parse snapUrl. %s", err)
	}
	InitSnapClient(snapUrl)
	catalog, err := GetSnapMetrics()
	if err != nil {
		log.Fatal(err)
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	controllerUrl := url.URL{Scheme: "ws", Host: *addr, Path: fmt.Sprintf("/socket/%s/%d", *nodeName, Version)}
	conn, err := connect(controllerUrl)
	if err != nil {
		log.Fatalf("unable to connect to server on url: %s", controllerUrl.String())
	}

	//create new session, allow 1000 events to be queued in the writeQueue before Emit() blocks.
	sess := session.NewSession(conn, 1000)
	sess.On("disconnect", func() {
		// on disconnect, reconnect.
		conn, err := connect(controllerUrl)
		for err != nil {
			time.Sleep(2 * time.Second)
			conn, err = connect(controllerUrl)
		}
		sess.Conn = conn
		go sess.Start()
	})

	sess.On("heartbeat", func(body []byte) {
		log.Debugf("recieved heartbeat event. %s", body)
	})
	go sess.Start()
	//send our MetricCatalog
	body, err := json.Marshal(catalog)
	if err != nil {
		log.Fatal(err)
	}
	e := &message.Event{Event: "catalog", Payload: body}
	sess.Emit(e)

	//periodically send an Updated Catalog.
	shutdownStart := make(chan struct{})
	go SendCatalog(sess, shutdownStart)

	//wait for interupt Signal.
	<-interrupt
	log.Info("interrupt")
	close(shutdownStart)
	sess.Close()
	return
}

func SendCatalog(sess *session.Session, shutdownStart chan struct{}) {
	ticker := time.NewTicker(time.Minute * 5)
	for {
		select {
		case <-shutdownStart:
			return
		case <-ticker.C:
			catalog, err := GetSnapMetrics()
			if err != nil {
				log.Error(err)
				continue
			}
			body, err := json.Marshal(catalog)
			if err != nil {
				log.Error(err)
				continue
			}
			e := &message.Event{Event: "catalog", Payload: body}
			sess.Emit(e)

		}
	}
}
