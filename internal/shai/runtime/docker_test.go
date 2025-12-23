package shai_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/client"
)

// isDockerAvailable attempts to connect to Docker using FromEnv first, then common socket paths.
func isDockerAvailable() bool {
	// First try client.FromEnv which works on all platforms
	if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, pingErr := cli.Ping(ctx)
		cancel()
		cli.Close()
		if pingErr == nil {
			return true
		}
	}

	// Fallback to common Unix socket paths
	socketPaths := []string{
		"unix:///var/run/docker.sock",                                     // Linux default
		"unix://" + os.Getenv("HOME") + "/.docker/run/docker.sock",        // Docker Desktop on macOS
		"unix:///Users/" + os.Getenv("USER") + "/.docker/run/docker.sock", // Alternative macOS path
	}

	for _, socketPath := range socketPaths {
		cli, err := client.NewClientWithOpts(
			client.WithHost(socketPath),
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, pingErr := cli.Ping(ctx)
		cancel()
		cli.Close()

		if pingErr == nil {
			os.Setenv("DOCKER_HOST", socketPath)
			return true
		}
	}

	return false
}

func requireDockerAvailable(t *testing.T) {
	t.Helper()
	if !isDockerAvailable() {
		t.Fatalf("Docker is required for this test; ensure Docker or Podman is running and reachable")
	}
}
