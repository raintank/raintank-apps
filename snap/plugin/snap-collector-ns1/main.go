package main

import (
	// Import the snap plugin library
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	// Import our collector plugin implementation
	"github.com/raintank/raintank-apps/snap/plugin/snap-collector-ns1/ns1"
)

const (
	pluginName    = "ns1"
	pluginVersion = 4
)

func main() {
	plugin.StartCollector(new(ns1.Ns1), pluginName, pluginVersion, plugin.ConcurrencyCount(50))
}
