package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	dockerApi "github.com/fsouza/go-dockerclient"
)

func GetInterfaceCobaltWeb(intPort int) (string, int, error) {
	log.Info("GetInterfaceCobaltWeb starting...")
	containerID := os.Getenv("HOSTNAME")

	hostIP := os.Getenv("HOST_IP")

	if containerID == "" || hostIP == "" {
		log.Error("Error: environment variable HOSTNAME or HOST_IPq is not set.")
		return "", -1, errors.New("Error: environment variable HOSTNAME or NOMAD_ADDR_cobalt is not set.")
	}
	port, e := GetAppExternalPort(hostIP, containerID, strconv.Itoa(intPort))
	if e != nil {
		msg := fmt.Sprintf("GetAppExternalPort encounter error: %v", e)
		return "", -1, errors.New(msg)

	}
	return hostIP, port, nil
}

// GetAppExternalPort get actual external port
func GetAppExternalPort(hostIP string, containerID string, appInternalPort string) (int, error) {
	log.Info("GetAppExternalPort starting...")

	client, err := dockerApi.NewClient("tcp://" + hostIP + ":2375")
	if err != nil {
		return -1, err
	}
	container, err := client.InspectContainer(containerID)
	if err != nil {
		return -1, err
	}

	portMaps := container.NetworkSettings.Ports
	protocolOrders := []string{"http", "udp"}
	for _, protocol := range protocolOrders {
		portNameStr := (appInternalPort + "/" + protocol)
		portName := dockerApi.Port(portNameStr)
		if list, ok := portMaps[portName]; ok {
			for _, ipAndPort := range list {
				if hostIP == ipAndPort.HostIP {
					port, e := strconv.Atoi(ipAndPort.HostPort)
					if e != nil {
						log.Errorf("GetAppExternalPort[services/docker] cannot convert port %v to int. %v", ipAndPort.HostPort, e.Error())
						continue
					}
					return port, nil
				}
			}
		}
	}
	return -1, errors.New("GetAppExternalPort[services/docker] Error: cannot find suitable port.")
}
