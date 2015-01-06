// Package utils provides some utility methods
package utils

import (
	"os"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	logging "github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("utils")
	// Refresh contains the global lock to sync configuration refresh.
	Refresh ConfigRefresh
)

// ConfigRefresh stores refresh synchronization data
type ConfigRefresh struct {
	Mu          sync.Mutex
	IsTriggered bool
	timer       *time.Timer
}

/*
EndPoint returns the first non empty string by evaluating:
	1. flag
	2. the environment variable specified by envVar
	3. sensibleDefault
*/
func EndPoint(sensibleDefault string, flag string, envVar string) string {
	if flag != "" {
		return flag
	} else if os.Getenv(envVar) != "" {
		return os.Getenv(envVar)
	} else {
		return sensibleDefault
	}
}

// RegisterDockerEventListener registers function as event listener with docker.
// laziness defines how many seconds to wait, after an event is received, until a refresh is triggered
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

	log.Info("Listening to docker events...")
	for {
		event := <-events

		if event == nil {
			continue
		}

		if event.Status == "start" || event.Status == "stop" || event.Status == "die" {
			log.Debug("Received event %s for container %s", event.Status, event.ID[:12])

			Refresh.Mu.Lock()
			if !Refresh.IsTriggered {
				log.Info("Triggering refresh in %d seconds", laziness)
				Refresh.timer = time.AfterFunc(time.Duration(laziness)*time.Second, function)
				Refresh.IsTriggered = true
			}
			Refresh.Mu.Unlock()
		}
	}
}

/*
TimeTrack can be used to log method execution time:

	defer utils.TimeTrack(time.Now(), "Config generation")
*/
func TimeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Debug("%s took %s", name, elapsed)
}
