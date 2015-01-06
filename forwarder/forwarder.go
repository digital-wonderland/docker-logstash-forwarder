package forwarder

import (
	"encoding/json"
	"os"
	"os/exec"
	"time"

	"github.com/digital-wonderland/docker-logstash-forwarder/forwarder/config"
	"github.com/digital-wonderland/docker-logstash-forwarder/utils"
	docker "github.com/fsouza/go-dockerclient"
	logging "github.com/op/go-logging"
)

var (
	cmd     *exec.Cmd
	log     = logging.MustGetLogger("forwarder")
	running = false
)

func getConfig(logstashEndpoint string, configFile string) *config.LogstashForwarderConfig {
	if configFile != "" {
		config, err := config.NewFromFile(configFile)
		if err != nil {
			log.Fatalf("Unable to read logstash-forwarder config from %s: %s", configFile, err)
		}
		log.Info("Using logstash-forwarder config from %s as template", configFile)
		return config
	}
	return config.NewFromDefault(logstashEndpoint)
}

// TriggerRefresh refreshes the logstash-forwarder configuration and restarts it.
func TriggerRefresh(client *docker.Client, logstashEndpoint string, configFile string) {
	defer utils.TimeTrack(time.Now(), "Config generation")

	log.Debug("Generating configuration...")
	forwarderConfig := getConfig(logstashEndpoint, configFile)

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		log.Fatalf("Unable to retrieve container list from docker: %s", err)
	}

	log.Debug("Found %d containers:", len(containers))
	for i, c := range containers {
		log.Debug("%d. %s", i+1, c.ID)

		container, err := client.InspectContainer(c.ID)
		if err != nil {
			log.Fatalf("Unable to inspect container %s: %s", c.ID, err)
		}

		forwarderConfig.AddContainerLogFile(container)

		containerConfig, err := config.NewFromContainer(container)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Error("Unable to look for logstash-forwarder config in %s: %s", container.ID, err)
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
		log.Debug("Unable to MarshalIndent logstash-forwarder config: %s", err)
	}
	_, err = fo.Write(j)
	if err != nil {
		log.Fatalf("Unable to write logstash-forwarder config to %s: %s", configPath, err)
	}
	log.Info("Wrote logstash-forwarder config to %s", configPath)

	if running {
		log.Info("Waiting for logstash-forwarder to stop")
		// perhaps use SIGTERM first instead of just Kill()?
		//		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		if err := cmd.Process.Kill(); err != nil {
			log.Error("Unable to stop logstash-forwarder")
		}
		if _, err := cmd.Process.Wait(); err != nil {
			log.Error("Unable to wait for logstash-forwarder to stop: %s", err)
		}
		log.Info("Stopped logstash-forwarder")
	}
	cmd = exec.Command("logstash-forwarder", "-config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Unable to start logstash-forwarder: %s", err)
	}
	running = true
	log.Info("Starting logstash-forwarder...")
}
