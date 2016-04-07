package main

import (
	"os"
	// Import the snap plugin library
	"github.com/intelsdi-x/snap/control/plugin"
	// Import our collector plugin implementation
	"github.com/raintank/raintank-apps/snap/plugin/snap-collector-ns1/ns1"
)

func main() {
	// Define metadata about Plugin
	meta := ns1.Meta()

	// Start a collector
	plugin.Start(meta, new(ns1.Ns1), os.Args[1])
}
