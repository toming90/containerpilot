package discovery

import log "github.com/Sirupsen/logrus"

// ServiceBackend is an interface
// which all service discovery backends must implement
type ServiceBackend interface {
	SendHeartbeat(service *ServiceDefinition)
	CheckForUpstreamChanges(backendName string, backendTag string) bool
	MarkForMaintenance(service *ServiceDefinition)
	Deregister(service *ServiceDefinition)
	GetClient() interface{}
}

// ServiceDefinition is the concrete service structure that is
// registered with the service discovery backend.
type ServiceDefinition struct {
	ID        string
	Name      string
	Port      int
	TTL       int
	Tags      []string
	IPAddress string
}

// ServiceDiscoveryConfigHook parses a raw service discovery config
type ServiceDiscoveryConfigHook func(interface{}) (ServiceBackend, error)

var backends = []string{}
var discoveryHooks = map[string]ServiceDiscoveryConfigHook{}

// RegisterBackend registers a service discovery config hook for a config key
func RegisterBackend(name string, hook ServiceDiscoveryConfigHook) {
	log.Debugf("Service discovery hook: %s\n", name)
	// map: consul - register function
	discoveryHooks[name] = hook

	// a list, consul, etcd, etc
	backends = append(backends, name)
}

// GetBackends gets the list of registered backends
func GetBackends() []string {
	return backends
}

// GetConfigHook gets the registered hook for the backend if it exists
func GetConfigHook(name string) ServiceDiscoveryConfigHook {
	if hook, ok := discoveryHooks[name]; ok {
		return hook
	}
	return nil
}
