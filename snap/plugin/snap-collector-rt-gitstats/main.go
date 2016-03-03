package main

import (
	"os"
	// Import the snap plugin library
	"github.com/intelsdi-x/snap/control/plugin"
	// Import our collector plugin implementation
	"github.com/raintank/raintank-apps/snap/plugin/snap-collector-rt-gitstats/gitstats"
)

func main() {
	// Define metadata about Plugin
	meta := gitstats.Meta()

	// Start a collector
	plugin.Start(meta, new(gitstats.Gitstats), os.Args[1])
}
