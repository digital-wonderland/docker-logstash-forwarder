package main

import (
	"flag"
	"log"
	"sync"

	"github.com/digital-wonderland/docker-logstash-forwarder/forwarder"
	"github.com/digital-wonderland/docker-logstash-forwarder/utils"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	client           *docker.Client
	configFile       string
	dockerEndPoint   string
	laziness         int
	logstashEndPoint string
	wg               sync.WaitGroup
)

func initFlags() {
	flag.StringVar(&dockerEndPoint, "docker", "", "docker api endpoint - defaults to $DOCKER_HOST or unix:///var/run/docker.sock")
	flag.IntVar(&laziness, "lazyness", 5, "number of seconds to wait after an event before generating new configuration")
	flag.StringVar(&logstashEndPoint, "logstash", "", "logstash endpoint - defaults to $LOGSTASH_HOST or logstash:5043")
	flag.StringVar(&configFile, "config", "", "logstash-forwarder config")
	flag.Parse()
}

func main() {
	log.Printf("Starting up")

	initFlags()

	endpoint := getDockerEndpoint()

	d, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatalf("Unable to connect to docker at %s: %s", endpoint, err)
	}
	client = d
	version, err := client.Version()
	if err != nil {
		log.Fatalf("Unable to retrieve version information from docker: %s", err)
	}
	log.Printf("Connected to docker at %s (v%s)", endpoint, version.Get("Version"))

	generateConfig()
	utils.RegisterDockerEventListener(client, generateConfig, &wg, laziness)
	wg.Wait()

	log.Println("done")
}

func generateConfig() {
	log.Printf("Generating new configuration...")
	utils.Refresh.Mu.Lock()
	utils.Refresh.IsTriggered = false
	utils.Refresh.Mu.Unlock()
	forwarder.GenerateConfig(client, getLogstashEndpoint(), configFile)
}

func getDockerEndpoint() string {
	return utils.EndPoint("unix:///var/run/docker.sock", dockerEndPoint, "DOCKER_HOST")
}

func getLogstashEndpoint() string {
	return utils.EndPoint("logstash:5043", logstashEndPoint, "LOGSTASH_HOST")
}
