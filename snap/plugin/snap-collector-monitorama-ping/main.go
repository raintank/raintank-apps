package main

import (
	"os"
	// Import the snap plugin library
	"github.com/intelsdi-x/snap/control/plugin"
	// Import our collector plugin implementation
	"github.com/raintank/raintank-apps/snap/plugin/snap-collector-monitorama-ping/ping"
)

func main() {
	// Define metadata about Plugin
	meta := ping.Meta()

	// Start a collector
	plugin.Start(meta, new(ping.Ping), os.Args[1])
}
