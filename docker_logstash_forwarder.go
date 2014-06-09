package main

import (
	"flag"
	"log"
	"sync"
	"os"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	dockerEndPoint string
	wg             sync.WaitGroup
)

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
		}
	}
}

func initFlags() {
	flag.StringVar(&dockerEndPoint, "endPoint", "", "docker api endpoint - defaults to unix:///var/run/docker.sock")
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
	log.Printf("Connected to docker at %s", endpoint)

	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	log.Printf("Found %d containers", len(containers))

	for _, container := range containers {
		log.Println("ID: ", container.ID)
	}

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

