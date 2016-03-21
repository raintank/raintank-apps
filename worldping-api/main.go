package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/Unknwon/macaron"
	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/met/helper"
	"github.com/raintank/raintank-apps/worldping-api/api"
	"github.com/raintank/raintank-apps/worldping-api/sqlstore"
	"github.com/raintank/raintank-apps/worldping-api/task_client"
	"github.com/rakyll/globalconf"
)

var (
	logLevel int

	showVersion = flag.Bool("version", false, "print version string")

	confFile = flag.String("config", "/etc/raintank/worldping-api.ini", "configuration file path")

	addr   = flag.String("addr", "localhost:80", "http service address")
	dbPath = flag.String("db-path", "/tmp/worldping-api.sqlite", "sqlite DB path")

	taskServer = flag.String("task-server-addr", "http://localhost:80", "Task server address")

	statsEnabled = flag.Bool("stats-enabled", false, "enable statsd metrics")
	statsdAddr   = flag.String("statsd-addr", "localhost:8125", "statsd address")
	statsdType   = flag.String("statsd-type", "standard", "statsd type: standard or datadog")

	adminKey = flag.String("admin-key", "not_very_secret_key", "Admin Secret Key")
)

func init() {
	flag.IntVar(&logLevel, "log-level", 2, "log level. 0=TRACE|1=DEBUG|2=INFO|3=WARN|4=ERROR|5=CRITICAL|6=FATAL")
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

	log.NewLogger(0, "console", fmt.Sprintf(`{"level": %d, "formatting":true}`, logLevel))
	// workaround for https://github.com/grafana/grafana/issues/4055
	switch logLevel {
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

	hostname, _ := os.Hostname()

	stats, err := helper.New(*statsEnabled, *statsdAddr, *statsdType, "worldping-api", strings.Replace(hostname, ".", "_", -1))
	if err != nil {
		log.Fatal(4, "failed to initialize statsd. %s", err)
	}

	// initialize DB
	sqlstore.NewEngine(*dbPath)

	// init taskServer client
	if err := task_client.Init(*taskServer, *adminKey, false); err != nil {
		log.Fatal(4, "Failed in init task client. %s", err)
	}

	m := macaron.Classic()
	//m.Use(macaron.Logger())
	m.Use(macaron.Renderer())

	api.Init(m, *adminKey, stats)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log.Info("starting up")
	// define our own listner so we can call Close on it
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(4, "failed to listen on %s: %s", *addr, err)
	}
	done := make(chan struct{})
	go handleShutdown(done, interrupt, l)
	log.Info("%s", http.Serve(l, m))
	<-done
}

func handleShutdown(done chan struct{}, interrupt chan os.Signal, l net.Listener) {
	<-interrupt
	log.Info("shutdown started.")
	l.Close()
	// handle any shutdown tasks here.
	close(done)
}
