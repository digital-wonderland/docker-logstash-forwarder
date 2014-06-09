package main

import (
	"os"
)

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
