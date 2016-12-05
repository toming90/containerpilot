package main // import "github.com/toming90/containerpilot"

import (
	"runtime"

	"github.com/toming90/containerpilot/core"

	log "github.com/Sirupsen/logrus"

	// Import backends so that they initialize
	_ "github.com/toming90/containerpilot/discovery/consul"
	_ "github.com/toming90/containerpilot/discovery/etcd"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

// Main executes the containerpilot CLI
func main() {
	// make sure we use only a single CPU so as not to cause
	// contention on the main application
	runtime.GOMAXPROCS(1)

	app, configErr := core.LoadApp()
	if configErr != nil {
		log.Fatal(configErr)
	}
	app.Run() // Blocks forever
}
