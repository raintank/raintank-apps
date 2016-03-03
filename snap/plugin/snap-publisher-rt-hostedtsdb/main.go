package main

import (
	"os"
	// Import the snap plugin library
	"github.com/intelsdi-x/snap/control/plugin"
	// Import our collector plugin implementation
	"github.com/raintank/raintank-apps/snap/plugin/snap-publisher-rt-hostedtsdb/hostedtsdb"
)

func main() {
	// Define metadata about Plugin
	meta := hostedtsdb.Meta()

	// Start a collector
	plugin.Start(meta, new(hostedtsdb.HostedtsdbPublisher), os.Args[1])
}
