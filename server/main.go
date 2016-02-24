// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/Unknwon/macaron"
	"github.com/op/go-logging"
	"github.com/raintank/raintank-apps/server/api"
	"github.com/raintank/raintank-apps/server/sqlstore"
	"github.com/rakyll/globalconf"
)

var log = logging.MustGetLogger("default")

var (
	showVersion = flag.Bool("version", false, "print version string")
	logLevel    = flag.Int("log-level", 5, "log level. 5=DEBUG|4=INFO|3=NOTICE|2=WARNING|1=ERROR|0=CRITICAL")
	confFile    = flag.String("config", "/etc/raintank/controller.ini", "configuration file path")

	addr   = flag.String("addr", "localhost:8081", "http service address")
	dbPath = flag.String("db-path", "/tmp/controller.sqlite", "sqlite DB path")
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

	// initialize DB
	sqlstore.NewEngine(*dbPath)

	m := macaron.Classic()
	m.Use(macaron.Logger())
	m.Use(macaron.Renderer())

	api.InitRoutes(m)

	log.Info("starting up")
	log.Fatal(http.ListenAndServe(*addr, m))
}
