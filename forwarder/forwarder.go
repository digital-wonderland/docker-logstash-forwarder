package forwarder

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/digital-wonderland/docker-logstash-forwarder/forwarder/config"
	"github.com/digital-wonderland/docker-logstash-forwarder/utils"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	cmd     *exec.Cmd
	running = false
)

func getConfig(logstashEndpoint string, configFile string) *config.LogstashForwarderConfig {
	if configFile != "" {
		config, err := config.NewFromFile(configFile)
		if err != nil {
			log.Fatalf("Unable to read logstash-forwarder config from %s: %s", configFile, err)
		}
		log.Printf("Using logstash-forwarder config from %s as template", configFile)
		return config
	}
	return config.NewFromDefault(logstashEndpoint)
}

// Refresh logstash-forwarder.
// The configuration is either initialized from configFile or the default is used (only important for certificate configuration).
func TriggerRefresh(client *docker.Client, logstashEndpoint string, configFile string) {
	defer utils.TimeTrack(time.Now(), "Config generation")

	log.Println("Generating configuration...")
	forwarderConfig := getConfig(logstashEndpoint, configFile)

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		log.Fatalf("Unable to retrieve container list from docker: %s", err)
	}

	log.Printf("Found %d containers:", len(containers))
	for i, c := range containers {
		log.Printf("%d. %s", i+1, c.ID)

		container, err := client.InspectContainer(c.ID)
		if err != nil {
			log.Fatalf("Unable to inspect container %s: %s", c.ID, err)
		}

		forwarderConfig.AddContainerLogFile(container)

		containerConfig, err := config.NewFromContainer(container)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Unable to look for logstash-forwarder config in %s: %s", container.ID, err)
			}
		} else {
			for _, file := range containerConfig.Files {
				file.Fields["host"] = container.Config.Hostname
				forwarderConfig.Files = append(forwarderConfig.Files, file)
			}
		}
	}

	const configPath = "/tmp/logstash-forwarder.conf"
	fo, err := os.Create(configPath)
	if err != nil {
		log.Fatalf("Unable to open %s: %s", configPath, err)
	}
	defer fo.Close()

	j, err := json.MarshalIndent(forwarderConfig, "", "  ")
	if err != nil {
		log.Printf("Unable to MarshalIndent logstash-forwarder config: %s", err)
	}
	_, err = fo.Write(j)
	if err != nil {
		log.Fatalf("Unable to write logstash-forwarder config to %s: %s", configPath, err)
	}
	log.Printf("Wrote logstash-forwarder config to %s", configPath)

	if running {
		log.Println("Waiting for logstash-forwarder to stop")
		// perhaps use SIGTERM first instead of just Kill()?
		//		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Unable to stop logstash-forwarder")
		}
		if _, err := cmd.Process.Wait(); err != nil {
			log.Printf("Unable to wait for logstash-forwarder to stop: %s", err)
		}
		log.Printf("Stopped logstash-forwarder")
	}
	cmd = exec.Command("logstash-forwarder", "-config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("Unable to start logstash-forwarder: %s", err)
	}
	running = true
	log.Printf("Starting logstash-forwarder...")
}
