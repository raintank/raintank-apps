package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"

	"github.com/raintank/raintank-apps/task-server/api"
	"github.com/raintank/raintank-apps/task-server/event"
	"github.com/raintank/raintank-apps/task-server/manager"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
	"github.com/raintank/raintank-apps/task-server/taskserverconfig"
	"github.com/raintank/raintank-probe/publisher"
	"github.com/raintank/worldping-api/pkg/log"
	"github.com/rakyll/globalconf"
)

var (
	GitHash     = "(none)"
	showVersion = flag.Bool("version", false, "print version string")
	logLevel    = flag.Int("log-level", 2, "log level. 0=TRACE|1=DEBUG|2=INFO|3=WARN|4=ERROR|5=CRITICAL|6=FATAL")
	confFile    = flag.String("config", "/etc/raintank/task-server.ini", "configuration file path")

	addr            = flag.String("addr", "localhost:80", "http service address")
	dbType          = flag.String("db-type", "sqlite3", "Database type. sqlite3 or mysql")
	dbConnectString = flag.String("db-connect-str", "file:/tmp/task-server.db?cache=shared&mode=rwc&_loc=Local", "DSN to connect to DB. https://godoc.org/github.com/mattn/go-sqlite3#SQLiteDriver.Open or https://github.com/go-sql-driver/mysql#dsn-data-source-name")

	exchange    = flag.String("exchange", "events", "Rabbitmq Topic Exchange")
	rabbitmqUrl = flag.String("rabbitmq-url", "amqp://guest:guest@localhost:5672/", "rabbitmq Url")
	tsdbAddr    = flag.String("tsdb-url", "http://localhost:2003/", "address of raintank-apps server")

	adminKey = flag.String("admin-key", "not_very_secret_key", "Admin Secret Key")
)

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
	var nodeName, err = os.Hostname()
	if err != nil {
		log.Fatal(4, "failed to get hostname from OS.")
	}

	tsdbURL, err := url.Parse(*tsdbAddr)
	if err != nil {
		log.Fatal(4, "Invalid TSDB url.", err)
	}
	var tsdbAPIKey = "123"
	publisher.Init(tsdbURL, tsdbAPIKey, 5)
	taskserverconfig.ConfigSetup()
	taskserverconfig.ConfigProcess(nodeName)
	taskserverconfig.Start()

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
		fmt.Printf("task-server (built with %s, git hash %s)\n", runtime.Version(), GitHash)
		return
	}

	hostname, _ := os.Hostname()

	// initialize DB
	enableSqlLog := false
	if *logLevel >= int(log.DEBUG) {
		enableSqlLog = true
	}
	sqlstore.NewEngine(*dbType, *dbConnectString, enableSqlLog)

	// delete any stale agentSessions.
	if err := sqlstore.DeleteAgentSessionsByServer(hostname); err != nil {
		panic(err)
	}

	m := api.NewApi(*adminKey)

	err = event.Init(*rabbitmqUrl, *exchange)
	if err != nil {
		log.Fatal(4, "failed to init event PubSub. %s", err)
	}

	manager.Init()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log.Info("starting up")
	// define our own listner so we can call Close on it
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(4, err.Error())
	}
	done := make(chan struct{})
	go handleShutdown(done, interrupt, l)
	log.Info("%v", http.Serve(l, m))
	<-done
}

func handleShutdown(done chan struct{}, interrupt chan os.Signal, l net.Listener) {
	<-interrupt
	log.Info("shutdown started.")
	l.Close()
	api.ActiveSockets.CloseAll()
	close(done)
}
