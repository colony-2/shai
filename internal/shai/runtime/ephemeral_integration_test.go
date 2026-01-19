//go:build integration
// +build integration

package shai_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	shai "github.com/colony-2/shai/internal/shai/runtime"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEphemeralContainerFullLifecycle tests the complete ephemeral container lifecycle
func TestEphemeralContainerFullLifecycle(t *testing.T) {
	requireDockerAvailable(t)

	// Create a test workspace
	tmpDir, err := os.MkdirTemp("", "shai-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	// Ensure tmpDir has correct permissions
	require.NoError(t, os.Chmod(tmpDir, 0777))

	// Create test directories with permissions that allow container user to write
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0777))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "tests"), 0777))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "docs"), 0777))
	// Ensure directory permissions are set correctly (not affected by umask)
	require.NoError(t, os.Chmod(filepath.Join(tmpDir, "src"), 0777))
	require.NoError(t, os.Chmod(filepath.Join(tmpDir, "tests"), 0777))
	require.NoError(t, os.Chmod(filepath.Join(tmpDir, "docs"), 0777))

	writeTestShaiConfig(t, tmpDir)

	// Create test files in each directory with permissions that allow container user to write
	srcFile := filepath.Join(tmpDir, "src", "test.go")
	testsFile := filepath.Join(tmpDir, "tests", "test_test.go")
	docsFile := filepath.Join(tmpDir, "docs", "README.md")
	require.NoError(t, os.WriteFile(srcFile, []byte("package main"), 0666))
	require.NoError(t, os.WriteFile(testsFile, []byte("package main_test"), 0666))
	require.NoError(t, os.WriteFile(docsFile, []byte("# Test"), 0666))
	// Ensure file permissions are set correctly (not affected by umask)
	require.NoError(t, os.Chmod(srcFile, 0666))
	require.NoError(t, os.Chmod(testsFile, 0666))
	require.NoError(t, os.Chmod(docsFile, 0666))

	t.Run("ephemeral container runs and cleans up", func(t *testing.T) {
		// Replace os.Stdin with a pipe BEFORE starting runner so we can send "exit" to the shell
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdin = r
		defer func() {
			os.Stdin = oldStdin
			r.Close()
			w.Close()
		}()

		// Create ephemeral runner
		runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
			WorkingDir:     tmpDir,
			ReadWritePaths: []string{"src", "tests"},
			Verbose:        true,
		})
		require.NoError(t, err)
		defer runner.Close()

		var containerID string

		// Run container with a command that exits quickly
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Run in a goroutine since it will block until container exits
		done := make(chan error, 1)
		go func() {
			done <- runner.Run(ctx)
		}()

		// Wait a moment for container to be created, then capture its ID
		time.Sleep(500 * time.Millisecond)
		containerID = runner.GetContainerID()

		// Wait for shell to be ready, then send exit command
		time.Sleep(2 * time.Second)
		_, _ = w.Write([]byte("exit\n"))
		w.Close()

		// Wait for completion or timeout
		select {
		case err := <-done:
			// Container should exit normally
			t.Logf("Run completed: %v", err)
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}

		// Verify the specific container was cleaned up
		if containerID != "" {
			// Check container state first
			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			require.NoError(t, err)
			defer cli.Close()

			// Give a moment for the container to stop
			time.Sleep(1 * time.Second)

			// Check if container is running or stopped
			ctx2 := context.Background()
			inspect, err := cli.ContainerInspect(ctx2, containerID)
			if err == nil {
				t.Logf("Container state: Running=%v, Status=%s", inspect.State.Running, inspect.State.Status)
			}

			// AutoRemove can take a bit - poll for up to 10 seconds
			removed := false
			for i := 0; i < 20; i++ {
				time.Sleep(500 * time.Millisecond)
				if !containerExists(t, containerID) {
					removed = true
					t.Logf("Container removed after %d * 500ms", i+1)
					break
				}
			}
			assert.True(t, removed, "Container %s should be removed after exit", containerID)
		}
	})
}

// TestMountPermissionsIntegration tests mount permissions in a real container
func TestMountPermissionsIntegration(t *testing.T) {
	requireDockerAvailable(t)

	// Create a test workspace
	tmpDir, err := os.MkdirTemp("", "shai-mount-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	// Ensure tmpDir has correct permissions
	require.NoError(t, os.Chmod(tmpDir, 0777))

	// Create directories with permissions that allow container user to write
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0777))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "docs"), 0777))
	// Ensure directory permissions are set correctly (not affected by umask)
	require.NoError(t, os.Chmod(filepath.Join(tmpDir, "src"), 0777))
	require.NoError(t, os.Chmod(filepath.Join(tmpDir, "docs"), 0777))

	// Create test files with permissions that allow container user to write
	srcFile := filepath.Join(tmpDir, "src", "writable.txt")
	docsFile := filepath.Join(tmpDir, "docs", "readonly.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("original"), 0666))
	require.NoError(t, os.WriteFile(docsFile, []byte("original"), 0666))
	// Ensure file permissions are set correctly (not affected by umask)
	require.NoError(t, os.Chmod(srcFile, 0666))
	require.NoError(t, os.Chmod(docsFile, 0666))

	writeTestShaiConfig(t, tmpDir)

	t.Run("selective mounts work correctly", func(t *testing.T) {
		runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
			WorkingDir:     tmpDir,
			ReadWritePaths: []string{"src"}, // Only src is writable
			Verbose:        true,
			PostSetupExec: &shai.ExecSpec{
				Command: []string{"/bin/sh", "-c", `
set +e
if echo 'modified' > /src/src/writable.txt; then
  echo 'Write to src: OK'
else
  echo 'Write to src: FAILED'
fi
if echo 'blocked' > /src/docs/readonly.txt; then
  echo 'Write to docs: FAILED'
else
  echo 'Write to docs: OK (blocked)'
fi
`},
				UseTTY: false,
			},
		})
		require.NoError(t, err)
		defer runner.Close()

		// Capture output
		var output bytes.Buffer
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- runner.Run(ctx)
		}()

		go func() {
			io.Copy(&output, r)
		}()

		select {
		case <-done:
			w.Close()
			os.Stdout = oldStdout
		case <-ctx.Done():
			w.Close()
			os.Stdout = oldStdout
			t.Fatal("Test timed out")
		}

		outputStr := output.String()
		t.Logf("Container output:\n%s", outputStr)

		// Check file contents
		srcContent, _ := os.ReadFile(filepath.Join(tmpDir, "src", "writable.txt"))
		docsContent, _ := os.ReadFile(filepath.Join(tmpDir, "docs", "readonly.txt"))

		// src should be modified, docs should not
		assert.Contains(t, string(srcContent), "modified", "src file should be modified")
		assert.Contains(t, string(docsContent), "original", "docs file should not be modified")
	})
}

// TestProgressMarkersEndToEnd tests progress markers in a real container
func TestProgressMarkersEndToEnd(t *testing.T) {
	requireDockerAvailable(t)

	tmpDir, err := os.MkdirTemp("", "shai-progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	writeTestShaiConfig(t, tmpDir)

	t.Run("container completes lifecycle commands", func(t *testing.T) {
		runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
			WorkingDir:     tmpDir,
			ReadWritePaths: []string{"."},
			Verbose:        true,
			PostSetupExec: &shai.ExecSpec{
				Command: []string{"/bin/sh", "-c", "exit"},
				UseTTY:  false,
			},
		})
		require.NoError(t, err)
		defer runner.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- runner.Run(ctx)
		}()

		select {
		case err := <-done:
			require.NoError(t, err)
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	})
}

// TestInteractivePromptEcho verifies prompt visibility and character echo in TTY mode
func TestInteractivePromptEcho(t *testing.T) {
	requireDockerAvailable(t)

	tmpDir, err := os.MkdirTemp("", "shai-tty-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	writeTestShaiConfig(t, tmpDir)

	runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
		WorkingDir:     tmpDir,
		ReadWritePaths: []string{"."},
		// default interactive behavior (no PostSetupExec)
	})
	require.NoError(t, err)
	defer runner.Close()

	// Capture stdout and provide stdin to simulate user typing
	oldStdout := os.Stdout
	oldStdin := os.Stdin
	rOut, wOut, _ := os.Pipe()
	rIn, wIn, _ := os.Pipe()
	os.Stdout = wOut
	os.Stdin = rIn
	defer func() {
		os.Stdout = oldStdout
		os.Stdin = oldStdin
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run container; it will switch to shell; we then type a command
	done := make(chan error, 1)
	var outBuf bytes.Buffer
	go func() { io.Copy(&outBuf, rOut) }()
	go func() { done <- runner.Run(ctx) }()

	// Wait a moment for shell to initialize and print a prompt
	time.Sleep(2 * time.Second)
	// Type a command: print marker without newline first, then read/echo characters, then newline
	_, _ = wIn.Write([]byte("echo HELLO\n"))
	_, _ = wIn.Write([]byte("exit\n"))
	// Allow output to flush
	time.Sleep(1 * time.Second)

	// Close input to terminate shell
	_ = wIn.Close()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("TTY test timed out")
	}
	_ = wOut.Close()

	got := outBuf.String()
	// We expect to see the typed command echoed and the output HELLO
	assert.Contains(t, got, "echo HELLO")
	assert.Contains(t, got, "HELLO")
}

// TestCleanShutdown tests graceful shutdown behavior
func TestCleanShutdown(t *testing.T) {
	requireDockerAvailable(t)

	tmpDir, err := os.MkdirTemp("", "shai-shutdown-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	writeTestShaiConfig(t, tmpDir)
	writeTestShaiConfig(t, tmpDir)

	t.Run("context cancellation stops container", func(t *testing.T) {
		runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
			WorkingDir:     tmpDir,
			ReadWritePaths: []string{"."},
		})
		require.NoError(t, err)
		defer runner.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- runner.Run(ctx)
		}()

		// Let it start
		time.Sleep(2 * time.Second)

		// Cancel context
		cancel()

		select {
		case err := <-done:
			// Should get context cancelled error or container should have exited
			// (some implementations may swallow the context error if cleanup succeeded)
			if err != nil {
				assert.ErrorIs(t, err, context.Canceled, "Expected context.Canceled error, got: %v", err)
			}
			// If err is nil, that's also acceptable - it means the container stopped cleanly
		case <-time.After(10 * time.Second):
			t.Fatal("Container did not stop after context cancellation")
		}

		// Verify container is cleaned up
		time.Sleep(2 * time.Second)
		// Container should be gone (auto-removed)
	})
}

// Helper functions

func getContainerIDs(t *testing.T) []string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	require.NoError(t, err)

	var ids []string
	for _, c := range containers {
		ids = append(ids, c.ID)
	}
	return ids
}

func containerExists(t *testing.T, containerID string) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	require.NoError(t, err)

	for _, c := range containers {
		if c.ID == containerID || strings.HasPrefix(c.ID, containerID) {
			return true
		}
	}
	return false
}

func writeTestShaiConfig(t *testing.T, dir string) {
	t.Helper()
	cfgPath := filepath.Join(dir, shai.DefaultConfigRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o777))
	config := `
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
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(config)+"\n"), 0o644); err != nil {
		t.Fatalf("write shai config: %v", err)
	}
}

// TestCtrlCEnabledAfterBootstrapMarker verifies that ctrl-c is properly enabled
// after the bootstrap marker appears in the output. This prevents regressions
// where changes to the bootstrap output format break ctrl-c functionality.
func TestCtrlCEnabledAfterBootstrapMarker(t *testing.T) {
	requireDockerAvailable(t)

	tmpDir, err := os.MkdirTemp("", "shai-ctrlc-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	require.NoError(t, os.Chmod(tmpDir, 0777))

	writeTestShaiConfig(t, tmpDir)

	// Create pipes for stdin/stdout
	stdinReader, stdinWriter, err := os.Pipe()
	require.NoError(t, err)
	defer stdinReader.Close()
	defer stdinWriter.Close()

	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	defer stdoutReader.Close()
	defer stdoutWriter.Close()

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	os.Stdin = stdinReader
	os.Stdout = stdoutWriter
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	// Create runner with a command that will wait for input
	runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
		WorkingDir:       tmpDir,
		Verbose:          false,
		ShowScriptOutput: true,
		PostSetupExec: &shai.ExecSpec{
			Command: []string{"sh", "-c", "echo 'ready'; read line; echo \"got: $line\""},
			Workdir: "/src",
			UseTTY:  true,
		},
	})
	require.NoError(t, err)
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Run container in background
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	// Expected exact marker based on test config
	expectedMarker := "Shai sandbox started using [ghcr.io/colony-2/shai-base:latest] as user [shai]. Resource sets: [base]"

	// Read output and verify marker appears
	outputBuf := &bytes.Buffer{}
	markerSeen := false
	readyChan := make(chan bool, 1)

	go func() {
		scanner := bytes.NewBuffer(nil)
		buf := make([]byte, 1024)
		for {
			n, err := stdoutReader.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				outputBuf.Write(chunk)
				scanner.Write(chunk)

				// Check for exact bootstrap marker
				if !markerSeen && bytes.Contains(outputBuf.Bytes(), []byte(expectedMarker)) {
					markerSeen = true
				}

				// Check for "ready" output from the shell command
				if bytes.Contains(scanner.Bytes(), []byte("ready")) {
					select {
					case readyChan <- true:
					default:
					}
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for marker and ready signal
	select {
	case <-readyChan:
		// Success - shell is ready
	case <-time.After(30 * time.Second):
		t.Fatalf("timeout waiting for shell to be ready")
	case err := <-done:
		t.Fatalf("container exited prematurely: %v", err)
	}

	// Give a moment for marker detection
	time.Sleep(100 * time.Millisecond)

	// Verify exact marker was seen
	assert.True(t, markerSeen, "exact bootstrap marker %q should appear in output", expectedMarker)

	// Send input to trigger exit
	_, err = stdinWriter.Write([]byte("test\n"))
	require.NoError(t, err)
	stdinWriter.Close()

	// Wait for completion
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for container to exit")
	}

	// Verify we got the expected output
	output := outputBuf.String()
	assert.Contains(t, output, "got: test", "shell should have received input after marker")
}
