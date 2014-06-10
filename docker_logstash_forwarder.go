package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"sync"

	docker "github.com/fsouza/go-dockerclient"
	"fmt"
)

var (
	configFile     string
	dockerEndPoint string
	wg             sync.WaitGroup
)

type Network struct {
	Servers        []string `json:"servers"`
	SslCertificate string `json:"ssl certificate"`
}

type File struct {
	Paths []string `json:"paths"`
	Fields map[string]string `json:"fields"`
}

type LogstashForwarderConfig struct {
	Network Network `json:"network"`
	Files   []File `json:"files"`
}

func readLogstashForwarderConfig(path string) (*LogstashForwarderConfig, error) {
//	log.Printf("Reading logstash-forwarder config from %s", path)

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

func generateConfig(client *docker.Client) error {
	log.Println("Generating configuration...")
	globalConfig, err := readLogstashForwarderConfig(configFile)
	if err != nil {
		log.Fatalf("Unable to read logstash-forwarder config from %s: %s", configFile, err)
	}
	log.Printf("Using logstash-forwarder config from %s as template", configFile)

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		log.Fatalf("Unable to retrieve container list from docker: %s", err)
	}

	log.Printf("Found %d containers:", len(containers))
	for i, container := range containers {
		log.Printf("%d. %s", i+1, container.ID)

		containerConfig, err := getLogstashForwarderConfigForContainer(container.ID)
		if err != nil {
			log.Printf("No logstash-forwarder config found in %s: %s", container.ID, err)
		} else {
			log.Printf("Found logstash-forwarder config in %s", container.ID)

			hostname := getHostnameForContainer(container.ID)
			log.Printf("Hostname: %s", hostname)

			for _, file := range containerConfig.Files {
				log.Printf("Adding file %s of type %s", file.Paths, file.Fields["type"])
				file.Fields["host"] = hostname
				globalConfig.Files = append(globalConfig.Files, file)
			}
		}


	}


	log.Printf("Network: %s", globalConfig.Network.Servers)

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

func listenToDockerEvents(client *docker.Client) {
	wg.Add(1)
	defer wg.Done()

	events := make(chan *docker.APIEvents)
	defer close(events)

	err := client.AddEventListener((chan <- *docker.APIEvents)(events))
	if err != nil {
		log.Fatal("Unable to add docker event listener: %s", err)
	}
	defer client.RemoveEventListener(events)

	log.Println("Listening to docker events...")
	for {
		event := <-events

		if event == nil {
			continue
		}

		if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
			log.Printf("Received event %s for container %s", event.Status, event.ID[:12])
			generateConfig(client)
		}
	}
}

func initFlags() {
	flag.StringVar(&dockerEndPoint, "docker", "", "docker api endpoint - defaults to unix:///var/run/docker.sock")
	flag.StringVar(&configFile, "config", "/etc/logstash-forwarder.conf", "logstash-forwarder config")
	flag.Parse()
}

func main() {
	log.Printf("Starting up")

	initFlags()

	endpoint := getDockerEndpoint()

	client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatalf("Unable to connect to docker at %s: %s", endpoint, err)
	}
	version, err := client.Version()
	if err != nil {
		log.Fatalf("Unable to retrieve version information from docker: %s", err)
	}
	log.Printf("Connected to docker at %s (v%s)", endpoint, version.Get("Version"))

	generateConfig(client)
	listenToDockerEvents(client)
	wg.Wait()

	log.Println("done")
}

func getDockerEndpoint() string {
	defaultEndpoint := "unix:///var/run/docker.sock"
	if os.Getenv("DOCKER_HOST") != "" {
		defaultEndpoint = os.Getenv("DOCKER_HOST")
	}

	if dockerEndPoint != "" {
		defaultEndpoint = dockerEndPoint
	}

	return defaultEndpoint
}

