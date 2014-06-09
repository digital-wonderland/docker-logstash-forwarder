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

type Event struct {
	ContainerID string `json:"id"`
	Status      string `json:"status"`
	Image       string `json:"from"`
}

func listenToDockerEvents(client *docker.Client) {
	wg.Add(1)
	defer wg.Done()
	log.Println("Watching docker events")
	eventChan := getEvents()
	for {
		event := <-eventChan

		if event == nil {
			continue
		}

		if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
			log.Printf("Received event %s for container %s", event.Status, event.ContainerID[:12])
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

	log.Printf("Using docker endpoint: %s", endpoint)

	client, err := docker.NewClient(endpoint)

	if err != nil {
		log.Fatalf("Unable to parse %s: %s", endpoint, err)
	}

	imgs, _ := client.ListImages(true)

	client.

	log.Printf("Found %d images", len(imgs))

	for _, img := range imgs {
		log.Println("ID: ", img.ID)
		log.Println("Repository: ", img.Repository)
	}

	log.Println("Path: ", os.Getenv("PATH"))
	log.Println("Docker: ", os.Getenv("DOCKER_HOST"))

	log.Println("Listening to docker events...")
	listenToDockerEvents(client)
	wg.Wait()

	log.Println("done")
}

//func getEndpoint() string {
//	defaultEndpoint := "unix:///var/run/docker.sock"
//	if os.Getenv("DOCKER_HOST") != "" {
//		defaultEndpoint = os.Getenv("DOCKER_HOST")
//	}
//
//	if endPoint != "" {
//		defaultEndpoint = endPoint
//	}
//
//	return defaultEndpoint
//}

