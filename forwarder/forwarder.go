package forwarder

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	docker "github.com/fsouza/go-dockerclient"
)

var (
	cmd     *exec.Cmd
	running = false
)

type Network struct {
	Servers        []string `json:"servers"`
	SslCertificate string   `json:"ssl certificate"`
	SslKey         string   `json:"ssl key"`
	SslCa          string   `json:"ssl ca"`
	Timeout        int64    `json:"timeout"`
}

type File struct {
	Paths  []string          `json:"paths"`
	Fields map[string]string `json:"fields"`
}

type LogstashForwarderConfig struct {
	Network Network `json:"network"`
	Files   []File  `json:"files"`
}

func readLogstashForwarderConfig(path string) (*LogstashForwarderConfig, error) {
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

func generateDefaultConfig(logstashEndpoint string) *LogstashForwarderConfig {
	network := Network{
		Servers:        []string{logstashEndpoint},
		SslCertificate: "/etc/pki/tls/certs/logstash-forwarder.crt",
		SslKey:         "/etc/pki/tls/private/logstash-forwarder.key",
		SslCa:          "/etc/pki/tls/certs/logstash-forwarder.crt",
		Timeout:        15,
	}

	config := &LogstashForwarderConfig{
		Network: network,
		Files:   []File{},
	}

	return config
}

func addConfigForContainer(config *LogstashForwarderConfig, container *docker.Container) {
	id := container.ID
	file := File{}
	file.Paths = []string{fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", id, id)}
	file.Fields = make(map[string]string)
	file.Fields["type"] = "docker"
	file.Fields["docker.id"] = id
	file.Fields["docker.hostname"] = container.Config.Hostname
	file.Fields["docker.name"] = container.Name
	file.Fields["docker.image"] = container.Config.Image

	config.Files = append(config.Files, file)
}

func getLogstashForwarderConfig(logstashEndpoint string, configFile string) *LogstashForwarderConfig {
	if configFile != "" {
		config, err := readLogstashForwarderConfig(configFile)
		if err != nil {
			log.Fatalf("Unable to read logstash-forwarder config from %s: %s", configFile, err)
		}
		log.Printf("Using logstash-forwarder config from %s as template", configFile)
		return config
	} else {
		return generateDefaultConfig(logstashEndpoint)
	}
}

func GenerateConfig(client *docker.Client, logstashEndpoint string, configFile string) {

	log.Println("Generating configuration...")
	globalConfig := getLogstashForwarderConfig(logstashEndpoint, configFile)

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

		addConfigForContainer(globalConfig, container)

		containerConfig, err := getLogstashForwarderConfigForContainer(container.ID)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Unable to look for logstash-forwarer config in %s: %s", container.ID, err)
			}
		} else {
			for _, file := range containerConfig.Files {
				file.Fields["host"] = container.Config.Hostname
				globalConfig.Files = append(globalConfig.Files, file)
			}
		}
	}

	path := "/tmp/logstash-forwarder.conf"
	fo, err := os.Create(path)
	if err != nil {
		log.Fatalf("Unable to open %s: %s", path, err)
	}
	defer fo.Close()

	j, err := json.MarshalIndent(globalConfig, "", "  ")
	fo.Write(j)
	if err != nil {
		log.Fatalf("Unable to write logstash-forwarder config to %s: %s", path, err)
	}
	log.Printf("Wrote logstash-forwarder config to %s", path)

	if running {
		log.Println("Waiting for logstash-forwarder to stop")
		// perhaps use SIGTERM instead of Kill()?
		//		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Unable to stop logstash-forwarder")
		}
		if _, err := cmd.Process.Wait(); err != nil {
			log.Printf("Unable to wait for logstash-forwarder to stop: %s", err)
		}
		log.Printf("Stopped logstash-forwarder")
	}
	cmd = exec.Command("logstash-forwarder", "-config", "/tmp/logstash-forwarder.conf")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Printf("Unable to start logstash-forwarder: %s", err)
	}
	running = true
	log.Printf("Starting logstash-forwarder...")
}

func getLogstashForwarderConfigForContainer(id string) (*LogstashForwarderConfig, error) {
	containerDirectory := fmt.Sprintf("/var/lib/docker/btrfs/subvolumes/%s", id)
	path := fmt.Sprintf("%s/etc/logstash-forwarder.conf", containerDirectory)
	config, err := readLogstashForwarderConfig(path)
	if err != nil {
		return nil, err
	}
	log.Printf("Found logstash-forwarder config in %s", id)

	for _, file := range config.Files {
		log.Printf("Adding files %s of type %s", file.Paths, file.Fields["type"])
		for i, path := range file.Paths {
			file.Paths[i] = containerDirectory + path
		}
	}

	return config, nil
}
