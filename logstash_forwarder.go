package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

type Network struct {
	Servers        []string `json:"servers"`
	SslCertificate string   `json:"ssl certificate"`
	SslKey         string   `json:"ssl key"`
	SslCa          string   `json:"ssl ca"`
	Timeout        int64    `json:"timeout"`
}

type File struct {
	Paths  []string          `json:"paths"`
	Fields map[string]string `json:"fields"`
}

type LogstashForwarderConfig struct {
	Network Network `json:"network"`
	Files   []File  `json:"files"`
}

func readLogstashForwarderConfig(path string) (*LogstashForwarderConfig, error) {
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

func getLogstashEndpoint() string {
	return getEndPoint("logstash:5043", logstashEndPoint, "LOGSTASH_HOST")
}

func generateDefaultConfig() *LogstashForwarderConfig {
	config := new(LogstashForwarderConfig)

	config.Network.Servers = []string{getLogstashEndpoint()}
	config.Network.SslCertificate = "/etc/pki/tls/certs/logstash-forwarder.crt"
	config.Network.SslKey = "/etc/pki/tls/private/logstash-forwarder.key"
	config.Network.SslCa = "/etc/pki/tls/certs/logstash-forwarder.crt"
	config.Network.Timeout = 15

	config.Files = []File{}

	return config
}

func addConfigForContainer(config *LogstashForwarderConfig, id string, hostname string) {
	file := File{}
	file.Paths = []string{fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", id, id)}
	file.Fields = make(map[string]string)
	file.Fields["type"] = "docker"
	file.Fields["docker.id"] = id
	file.Fields["docker.hostname"] = hostname

	config.Files = append(config.Files, file)
}

func getLogstashForwarderConfig() *LogstashForwarderConfig {
	if configFile != "" {
		config, err := readLogstashForwarderConfig(configFile)
		if err != nil {
			log.Fatalf("Unable to read logstash-forwarder config from %s: %s", configFile, err)
		}
		log.Printf("Using logstash-forwarder config from %s as template", configFile)
		return config
	} else {
		return generateDefaultConfig()
	}
}

func generateConfig(client *docker.Client) error {
	log.Println("Generating configuration...")
	globalConfig := getLogstashForwarderConfig()

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		log.Fatalf("Unable to retrieve container list from docker: %s", err)
	}

	log.Printf("Found %d containers:", len(containers))
	for i, container := range containers {
		log.Printf("%d. %s", i+1, container.ID)

		hostname := strings.Trim(getHostnameForContainer(container.ID), " \n")
		addConfigForContainer(globalConfig, container.ID, hostname)

		containerConfig, err := getLogstashForwarderConfigForContainer(container.ID)
		if err != nil {
			log.Printf("No logstash-forwarder config found in %s: %s", container.ID, err)
		} else {
			log.Printf("Found logstash-forwarder config in %s", container.ID)

			for _, file := range containerConfig.Files {
				log.Printf("Adding file %s of type %s", file.Paths, file.Fields["type"])
				file.Fields["host"] = hostname
				globalConfig.Files = append(globalConfig.Files, file)
			}
		}

	}

	log.Printf("Network: %s", globalConfig.Network.Servers)

	// print to /etc/logstash-forwarder.conf
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(globalConfig)

	return nil
}

func getLogstashForwarderConfigForContainer(id string) (*LogstashForwarderConfig, error) {
	containerDirectory := fmt.Sprintf("/var/lib/docker/btrfs/subvolumes/%s", id)
	path := fmt.Sprintf("%s/etc/logstash-forwarder.conf", containerDirectory)
	config, err := readLogstashForwarderConfig(path)
	if err != nil {
		return nil, err
	}

	for _, file := range config.Files {
		for i, path := range file.Paths {
			file.Paths[i] = containerDirectory + path
		}
	}

	return config, nil
}

func getHostnameForContainer(id string) string {
	path := fmt.Sprintf("/var/lib/docker/containers/%s/hostname", id)
	hostname, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Unable to read hostname of container %s from %s: %s", id, path, err)
	}
	return string(hostname[:])
}
