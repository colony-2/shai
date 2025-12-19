//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/colony-2/shai/pkg/shai"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testImage = "ghcr.io/colony-2/shai-base:latest"

// TestMain runs before all tests in this package to pre-pull required Docker images
func TestMain(m *testing.M) {

	fmt.Printf("Pre-pulling Docker image %s (this may take a while)...\n", testImage)
	if err := pullDockerImage(testImage); err != nil {
		fmt.Printf("Warning: failed to pre-pull image %s: %v\n", testImage, err)
		fmt.Println("Tests will attempt to pull the image themselves")
	} else {
		fmt.Printf("Successfully pulled image %s\n", testImage)
	}

	os.Exit(m.Run())
}

// pullDockerImage pulls a Docker image without timeout, reporting progress to stdout
func pullDockerImage(image string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	ctx := context.Background()

	// Check if image already exists
	if _, _, err := cli.ImageInspectWithRaw(ctx, image); err == nil {
		fmt.Printf("Image %s already exists locally\n", image)
		return nil
	}

	// Pull the image
	fmt.Printf("Pulling image %s...\n", image)
	reader, err := cli.ImagePull(ctx, image, imagetypes.PullOptions{})
	if err != nil {
		return fmt.Errorf("image pull: %w", err)
	}
	defer reader.Close()

	// Copy output to stdout to show progress
	if _, err := io.Copy(os.Stdout, reader); err != nil {
		return fmt.Errorf("read pull output: %w", err)
	}

	return nil
}

// TestCLI_EphemeralShell_StartsAndEchoes runs the shai CLI, starts a shell, echoes output, and exits
func TestCLI_EphemeralShell_StartsAndEchoes(t *testing.T) {
	if !dockerAvailable(t) {
		t.Skip("Docker not available")
	}

	tmp := t.TempDir()
	shaiCfg := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
resources:
  base: {}
apply:
  - path: ./
    resources:
      - base
`
	cfgPath := filepath.Join(tmp, shai.DefaultConfigRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o755))
	require.NoError(t, os.WriteFile(cfgPath, []byte(strings.TrimSpace(shaiCfg)+"\n"), 0o644))

	// Build CLI binary in a temp location to avoid races
	bin := filepath.Join(tmp, "shai_bin")
	build := exec.Command("go", "build", "-o", bin, ".")
	wd, err := os.Getwd()
	require.NoError(t, err)
	build.Dir = wd
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", string(out))

	// Run the CLI with stdin that immediately exits
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-rw", ".", "-no-tty", "--", "sh", "-c", "echo HELLO")
	cmd.Dir = tmp
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	got := stdout.String() + stderr.String()

	// Check that HELLO appears in output from the post-setup exec command
	assert.Contains(t, got, "HELLO", "CLI output should contain HELLO from exec command")

	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)
}

// dockerAvailable tries to ping Docker; returns true if reachable
func dockerAvailable(t *testing.T) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := cli.Ping(ctx); err == nil {
			_ = cli.Close()
			return true
		}
		_ = cli.Close()
	}
	// Try common sockets
	sockets := []string{
		"unix:///var/run/docker.sock",
		"unix://" + os.Getenv("HOME") + "/.docker/run/docker.sock",
	}
	for _, s := range sockets {
		cli, err := client.NewClientWithOpts(client.WithHost(s), client.WithAPIVersionNegotiation())
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err = cli.Ping(ctx)
		cancel()
		_ = cli.Close()
		if err == nil {
			return true
		}
	}
	return false
}
