package main

import (
	// Import the snap plugin library
	"github.com/intelsdi-x/snap-plugin-lib-go/v1/plugin"
	// Import our publisher plugin implementation
	"github.com/raintank/raintank-apps/snap/plugin/snap-publisher-rt-hostedtsdb/hostedtsdb"
)

const (
	pluginName    = "rt-hostedtsdb"
	pluginVersion = 2
)

func main() {
	plugin.StartPublisher(new(hostedtsdb.HostedtsdbPublisher), pluginName, pluginVersion, plugin.ConcurrencyCount(1000))
}
