package main

import (
	"os"
	// Import the snap plugin library
	"github.com/intelsdi-x/snap/control/plugin"
	// Import our collector plugin implementation
	"github.com/raintank/raintank-apps/snap/plugin/snap-collector-voxter/voxter"
)

func main() {
	// Define metadata about Plugin
	meta := voxter.Meta()

	// Start a collector
	plugin.Start(meta, new(voxter.Voxter), os.Args[1])
}
