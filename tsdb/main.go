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

	"github.com/Unknwon/macaron"
	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/met/helper"
	"github.com/raintank/raintank-apps/tsdb/api"
	"github.com/raintank/raintank-apps/tsdb/graphite"
	"github.com/raintank/raintank-apps/tsdb/metric_publish"
	"github.com/rakyll/globalconf"
)

var (
	GitHash     = "(none)"
	showVersion = flag.Bool("version", false, "print version string")
	logLevel    = flag.Int("log-level", 2, "log level. 0=TRACE|1=DEBUG|2=INFO|3=WARN|4=ERROR|5=CRITICAL|6=FATAL")
	confFile    = flag.String("config", "/etc/raintank/tsdb.ini", "configuration file path")

	metricTopic    = flag.String("metric-topic", "metrics", "NSQ topic")
	nsqdAddr       = flag.String("nsqd-addr", "localhost:4150", "nsqd service address")
	publishMetrics = flag.Bool("publish-metrics", false, "enable metric publishing")

	addr = flag.String("addr", "localhost:80", "http service address")

	statsEnabled = flag.Bool("stats-enabled", false, "enable statsd metrics")
	statsdAddr   = flag.String("statsd-addr", "localhost:8125", "statsd address")
	statsdType   = flag.String("statsd-type", "standard", "statsd type: standard or datadog")

	graphiteUrl = flag.String("graphite-url", "http://localhost:8080", "graphite-api address")

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
		fmt.Printf("tsdb (built with %s, git hash %s)\n", runtime.Version(), GitHash)
		return
	}

	hostname, _ := os.Hostname()

	stats, err := helper.New(*statsEnabled, *statsdAddr, *statsdType, "raintank_tsdb", strings.Replace(hostname, ".", "_", -1))
	if err != nil {
		log.Fatal(4, "failed to initialize statsd. %s", err)
	}

	metric_publish.Init(stats, *metricTopic, *nsqdAddr, *publishMetrics)

	m := macaron.Classic()
	m.Use(macaron.Logger())
	m.Use(macaron.Renderer())

	api.InitRoutes(m, *adminKey)

	if err := graphite.Init(*graphiteUrl); err != nil {
		log.Fatal(4, err.Error())
	}

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
	err = http.Serve(l, m)
	if err != nil {
		log.Info(err.Error())
	}
	<-done
}

func handleShutdown(done chan struct{}, interrupt chan os.Signal, l net.Listener) {
	<-interrupt
	log.Info("shutdown started.")
	l.Close()
	close(done)
}
