package shai

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

const testImage = "ghcr.io/colony-2/shai-base:latest"

func TestMain(m *testing.M) {
	// Force verbose mode during tests so setup logs remain visible.
	_ = os.Setenv("SHAI_FORCE_VERBOSE", "1")

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
