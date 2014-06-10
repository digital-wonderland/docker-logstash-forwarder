package main

import (
	"flag"
	"log"
	"os"
	"sync"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	configFile       string
	dockerEndPoint   string
	logstashEndPoint string
	wg               sync.WaitGroup
)

func listenToDockerEvents(client *docker.Client) {
	wg.Add(1)
	defer wg.Done()

	events := make(chan *docker.APIEvents)
	defer close(events)

	err := client.AddEventListener((chan<- *docker.APIEvents)(events))
	if err != nil {
		log.Fatalf("Unable to add docker event listener: %s", err)
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
	flag.StringVar(&logstashEndPoint, "logstash", "", "logstash endpoint - defaults to logstash:5043")
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
	return getEndPoint("unix:///var/run/docker.sock", dockerEndPoint, "DOCKER_HOST")
}

func getEndPoint(sensibleDefault string, flag string, envVar string) string {
	if flag != "" {
		return flag
	} else if os.Getenv(envVar) != "" {
		return os.Getenv(envVar)
	} else {
		return sensibleDefault
	}
}
