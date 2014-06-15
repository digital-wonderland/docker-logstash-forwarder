package utils

import (
	"log"
	"os"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	Refresh ConfigRefresh
)

type ConfigRefresh struct {
	Mu          sync.Mutex
	IsTriggered bool
	timer       *time.Timer
}

func EndPoint(sensibleDefault string, flag string, envVar string) string {
	if flag != "" {
		return flag
	} else if os.Getenv(envVar) != "" {
		return os.Getenv(envVar)
	} else {
		return sensibleDefault
	}
}

func RegisterDockerEventListener(client *docker.Client, function func(), wg *sync.WaitGroup, laziness int) {
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

			Refresh.Mu.Lock()
			if !Refresh.IsTriggered {
				log.Printf("Triggering refresh in %d seconds", laziness)
				Refresh.timer = time.AfterFunc(time.Duration(laziness)*time.Second, function)
				Refresh.IsTriggered = true
			}
			Refresh.Mu.Unlock()
		}
	}
}

func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}
