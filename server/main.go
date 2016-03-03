// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

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
	"github.com/op/go-logging"
	"github.com/raintank/met/helper"
	"github.com/raintank/raintank-apps/server/api"
	"github.com/raintank/raintank-apps/server/metric_publish"
	"github.com/raintank/raintank-apps/server/sqlstore"
	"github.com/rakyll/globalconf"
)

var log = logging.MustGetLogger("default")

var (
	showVersion = flag.Bool("version", false, "print version string")
	logLevel    = flag.Int("log-level", 5, "log level. 5=DEBUG|4=INFO|3=NOTICE|2=WARNING|1=ERROR|0=CRITICAL")
	confFile    = flag.String("config", "/etc/raintank/controller.ini", "configuration file path")

	metricTopic    = flag.String("metric-topic", "metrics", "NSQ topic")
	nsqdAddr       = flag.String("nsqd-addr", "localhost:4150", "nsqd service address")
	publishMetrics = flag.Bool("publish-metrics", false, "enable metric publishing")

	addr   = flag.String("addr", "localhost:8081", "http service address")
	dbPath = flag.String("db-path", "/tmp/controller.sqlite", "sqlite DB path")

	statsEnabled = flag.Bool("stats-enabled", false, "enable statsd metrics")
	statsdAddr   = flag.String("statsd-addr", "localhost:8125", "statsd address")
	statsdType   = flag.String("statsd-type", "standard", "statsd type: standard or datadog")
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

	logging.SetFormatter(logging.GlogFormatter)
	logging.SetLevel(logging.Level(*logLevel), "default")
	log.SetBackend(logging.AddModuleLevel(logging.NewLogBackend(os.Stdout, "", 0)))

	hostname, _ := os.Hostname()

	stats, err := helper.New(*statsEnabled, *statsdAddr, *statsdType, "raintank_apps", strings.Replace(hostname, ".", "_", -1))
	if err != nil {
		log.Fatalf("failed to initialize statsd. %s", err)
	}

	// initialize DB
	sqlstore.NewEngine(*dbPath)

	// delete any stale agentSessions.

	if err := sqlstore.DeleteAgentSessionsByServer(hostname); err != nil {
		panic(err)
	}

	metric_publish.Init(stats, *metricTopic, *nsqdAddr, *publishMetrics)

	m := macaron.Classic()
	m.Use(macaron.Logger())
	m.Use(macaron.Renderer())

	api.InitRoutes(m)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log.Info("starting up")
	// define our own listner so we can call Close on it
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan struct{})
	go handleShutdown(done, interrupt, l)
	log.Info(http.Serve(l, m))
	<-done
}

func handleShutdown(done chan struct{}, interrupt chan os.Signal, l net.Listener) {
	<-interrupt
	log.Info("shutdown started.")
	l.Close()
	api.ActiveSockets.CloseAll()
	close(done)
}
