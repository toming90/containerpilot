package main

import (
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
)

func TestMainContainerpilot(t *testing.T) {
	t.Skip()
	log.Debug("TestMain[main_test.go]: Start")
	defer log.Debug("TestMain[main_test.go]: Done")
	args := []string{"./containerpilot", "-config", "file:///Users/i851981/testing9/nomad-ui/cobalt_build/config/containerpilot.json", "echo helloWorld"}
	os.Args = args
	main()
}

func TestMainContainerpilotVersion(t *testing.T) {
	//	t.Skip()
	log.Debug("TestMain[main_test.go]: Start")
	defer log.Debug("TestMain[main_test.go]: Done")
	args := []string{"./containerpilot", "-version"}
	os.Args = args
	main()
}

//--config file://./testFiles/Containerpilot_case1.json echo hahaha
