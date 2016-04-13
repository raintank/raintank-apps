package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/met/helper"
	"github.com/raintank/raintank-apps/task-server/api"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
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

	statsEnabled = flag.Bool("stats-enabled", false, "enable statsd metrics")
	statsdAddr   = flag.String("statsd-addr", "localhost:8125", "statsd address")
	statsdType   = flag.String("statsd-type", "standard", "statsd type: standard or datadog")

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

	stats, err := helper.New(*statsEnabled, *statsdAddr, *statsdType, "raintank_apps", strings.Replace(hostname, ".", "_", -1))
	if err != nil {
		log.Fatal(4, "failed to initialize statsd. %s", err)
	}

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

	m := api.NewApi(*adminKey, stats)

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
