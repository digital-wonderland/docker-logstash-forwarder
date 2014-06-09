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
	log.Println("Listening to docker events")

	events := make(chan *docker.APIEvents)
	defer close(events)

	err := client.AddEventListener((chan<- *docker.APIEvents)(events))
	if err != nil {
		log.Fatal("Unable to add docker event listener: %s", err)
	}
	defer client.RemoveEventListener(events)

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

	log.Printf("Using docker endpoint: %s", endpoint)

	client, err := docker.NewClient(endpoint)

	if err != nil {
		log.Fatalf("Unable to parse %s: %s", endpoint, err)
	}

	//	imgs, _ := client.ListImages(true)
	//	imgs, _ := client.ListContainers(docker.ListContainersOptions{})
	imgs, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	log.Printf("Found %d images", len(imgs))

	for _, img := range imgs {
		log.Println("ID: ", img.ID)
		//		log.Println("Repository: ", img.Repository)
	}

	log.Println("Path: ", os.Getenv("PATH"))
	log.Println("Docker: ", os.Getenv("DOCKER_HOST"))

	log.Println("Listening to docker events...")
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

