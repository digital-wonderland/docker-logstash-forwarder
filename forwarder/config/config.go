// Package config implements functions to manipulate logstash-forwarder configs.
package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

// Network section of a configuration.
type Network struct {
	Servers        []string `json:"servers"`
	SslCertificate string   `json:"ssl certificate"`
	SslKey         string   `json:"ssl key"`
	SslCa          string   `json:"ssl ca"`
	Timeout        int64    `json:"timeout"`
}

// File section of a configuration.
type File struct {
	Paths  []string          `json:"paths"`
	Fields map[string]string `json:"fields"`
}

// LogstashForwarderConfig is the configs root structure.
type LogstashForwarderConfig struct {
	Network Network `json:"network"`
	Files   []File  `json:"files"`
}

// AddContainerLogFile adds the containers docker log file to this config.
func (config *LogstashForwarderConfig) AddContainerLogFile(container *docker.Container) {
	id := container.ID
	file := File{}
	file.Paths = []string{fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", id, id)}
	file.Fields = make(map[string]string)
	file.Fields["type"] = "docker"
	file.Fields["docker.id"] = id
	file.Fields["docker.hostname"] = container.Config.Hostname
	file.Fields["docker.name"] = container.Name
	file.Fields["docker.image"] = container.Config.Image

	config.Files = append(config.Files, file)
}

// NewFromFile returns a new config based on the file at path.
func NewFromFile(path string) (*LogstashForwarderConfig, error) {
	configFile, err := os.Open(path)
	defer configFile.Close()
	if err != nil {
		return nil, err
	}

	logstashConfig := new(LogstashForwarderConfig)

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&logstashConfig); err != nil {
		return nil, err
	}

	return logstashConfig, nil
}

// NewFromDefault returns a new default config.
func NewFromDefault(logstashEndpoint string) *LogstashForwarderConfig {
	network := Network{
		Servers:        []string{logstashEndpoint},
		SslCertificate: "/mnt/logstash-forwarder/logstash-forwarder.crt",
		SslKey:         "/mnt/logstash-forwarder/logstash-forwarder.key",
		SslCa:          "/mnt/logstash-forwarder/logstash-forwarder.crt",
		Timeout:        15,
	}

	config := &LogstashForwarderConfig{
		Network: network,
		Files:   []File{},
	}

	return config
}

// NewFromContainer returns a new config based on /etc/logstash-forwarder.conf within the container,
// if it exists.
func NewFromContainer(container *docker.Container) (*LogstashForwarderConfig, error) {
	path := fmt.Sprintf("/var/lib/docker/%s/mnt/%s/etc/logstash-forwarder.conf", container.Driver, container.ID)
	config, err := NewFromFile(path)
	if err != nil {
		return nil, err
	}
	log.Printf("Found logstash-forwarder config in %s", container.ID)

	for _, file := range config.Files {
		log.Printf("Adding files %s of type %s", file.Paths, file.Fields["type"])
		for i, path := range file.Paths {
			file.Paths[i] = calculateLogFilePath(container, path)
		}
	}
	return config, nil
}

func calculateLogFilePath(container *docker.Container, path string) string {
	for k, v := range container.Volumes {
		if strings.HasPrefix(path, k) {
			return v + strings.TrimPrefix(path, k)
		}
	}
	return fmt.Sprintf("/var/lib/docker/%s/mnt/%s%s", container.Driver, container.ID, path)
}
