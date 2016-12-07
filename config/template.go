package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"

	"github.com/toming90/containerpilot/utils"
)

// Environment is a map of environment variables to their values
type Environment map[string]string

func parseEnvironment(environ []string) Environment {
	env := make(Environment)
	if len(environ) == 0 {
		return env
	}
	for _, e := range environ {
		kv := strings.Split(e, "=")
		env[kv[0]] = kv[1]
	}

	//CUSTOMIZE
	//getting the ip in this order: nomad_addr,consul_ip(pass in), nomad_ip(pass_in), then getIP eth0.
	var hostAddress string
	var hostIP string

	hostAddress = os.Getenv("NOMAD_ADDR_cobalt")

	//it not triggered by NOMAD
	if "" == hostAddress {
		hostAddress = os.Getenv("CONSUL_IP")
		hostIP = hostAddress

	}
	//Try one more time.
	if "" == hostAddress {
		hostAddress = os.Getenv("NOMAD_IP")
		hostIP = hostAddress

	}

	//last resort getting from interface.
	if "" == hostAddress {
		if ipAddress, err := utils.GetIP([]string{"eth0"}); err != nil {
			hostIP = ipAddress
			hostAddress = hostIP
		}
	}

	if "" != hostAddress && "" == hostIP {
		hostIP = strings.Split(hostAddress, ":")[0]
	}

	log.Debugf("setting env host ip: %v", hostIP)
	// finnaly, set host ip
	if "" != hostIP {
		env["HOST_IP"] = hostIP
		os.Setenv("HOST_IP", hostIP)
		log.Debugf("parseEnvironment: set HOST_IP %v", hostIP)
	} else {
		log.Error("parseEnvironment: cannot get HOST_IP")
	}

	return env
}

// Template encapsulates a golang template
// and its associated environment variables.
type Template struct {
	Template *template.Template
	Env      Environment
}

func defaultValue(defaultValue, templateValue interface{}) string {
	if templateValue != nil {
		if str, ok := templateValue.(string); ok && str != "" {
			return str
		}
	}
	defaultStr, ok := defaultValue.(string)
	if !ok {
		return fmt.Sprintf("%v", defaultValue)
	}
	return defaultStr
}

// NewTemplate creates a Template parsed from the configuration
// and the current environment variables
func NewTemplate(config []byte) (*Template, error) {
	env := parseEnvironment(os.Environ())
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"default": defaultValue,
	}).Option("missingkey=zero").Parse(string(config))
	if err != nil {
		return nil, err
	}
	return &Template{
		Env:      env,
		Template: tmpl,
	}, nil
}

// Execute renders the template
func (c *Template) Execute() ([]byte, error) {
	var buffer bytes.Buffer
	if err := c.Template.Execute(&buffer, c.Env); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ApplyTemplate creates and renders a template from the given config template
func ApplyTemplate(config []byte) ([]byte, error) {
	template, err := NewTemplate(config)
	if err != nil {
		return nil, err
	}
	return template.Execute()
}
