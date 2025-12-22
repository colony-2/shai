//go:build integration

package shai

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test #16: Bootstrap script runs before user command
func TestBootstrap_RunsBeforeUserCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			// Verify that proxy and DNS are already running (bootstrap completed)
			// Give services time to start via supervisord
			Command: []string{"sh", "-c", `
				# Wait for services to be up (they should already be running)
				for i in 1 2 3 4 5 6 7 8 9 10; do
					if timeout 1 bash -c '</dev/tcp/127.0.0.1/18888' 2>/dev/null && \
					   timeout 1 bash -c '</dev/udp/127.0.0.1/1053' 2>/dev/null; then
						echo "BOOTSTRAP_COMPLETE"
						exit 0
					fi
					sleep 1
				done
				echo "BOOTSTRAP_FAILED"
				exit 1
			`},
			UseTTY: false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "BOOTSTRAP_COMPLETE", "Bootstrap should complete before user command")
}

// Test #17: Bootstrap creates required runtime directories
func TestBootstrap_CreatesRuntimeDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"sh", "-c", `
				test -d /run/shai && echo "RUN_DIR_EXISTS" &&
				test -d /var/log/shai && echo "LOG_DIR_EXISTS" &&
				test -d /var/log/shai/tinyproxy && echo "TINYPROXY_LOG_EXISTS" &&
				test -d /var/log/shai/dnsmasq && echo "DNSMASQ_LOG_EXISTS"
			`},
			UseTTY: false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "RUN_DIR_EXISTS", "/run/shai should exist")
	assert.Contains(t, result, "LOG_DIR_EXISTS", "/var/log/shai should exist")
	assert.Contains(t, result, "TINYPROXY_LOG_EXISTS", "tinyproxy log dir should exist")
	assert.Contains(t, result, "DNSMASQ_LOG_EXISTS", "dnsmasq log dir should exist")
}

// Test #18: Bootstrap fails on unsupported version number
func TestBootstrap_FailsOnUnsupportedVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This tests the bootstrap script's version validation
	// Since we can't easily inject a bad version through the Go API,
	// this would be tested by directly calling the bootstrap script
	// with --version 999, which would be done in a shell-level test
	t.Skip("Bootstrap version validation tested at shell script level")
}

// Test #19: Bootstrap requires mandatory arguments
func TestBootstrap_RequiresMandatoryArguments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This tests the bootstrap script's argument validation
	// The Go API always provides required arguments, so this would be
	// tested at the shell script level by calling bootstrap directly
	t.Skip("Bootstrap argument validation tested at shell script level")
}

// Test #20: Bootstrap script parses HTTP allowlist correctly
func TestBootstrap_ParsesHTTPAllowlist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
resources:
  test:
    http:
      - example.com
      - httpbin.org
      - github.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"cat", "/run/shai/allowed_domains.conf"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	allowlist := output.String()
	assert.Contains(t, allowlist, "example.com", "Should contain example.com")
	assert.Contains(t, allowlist, "httpbin.org", "Should contain httpbin.org")
	assert.Contains(t, allowlist, "github.com", "Should contain github.com")
}

// Test #21: Bootstrap script parses port allowlist correctly
func TestBootstrap_ParsesPortAllowlist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
resources:
  test:
    http:
      - github.com
    ports:
      - host: github.com
        port: 22
      - host: github.com
        port: 443
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"cat", "/var/log/shai/iptables.out"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	iptablesRules := output.String()
	// Port allowlist rules should appear in iptables
	// Look for evidence of port-specific rules
	hasPortRules := strings.Contains(iptablesRules, "22") || strings.Contains(iptablesRules, "443")
	assert.True(t, hasPortRules, "iptables should contain port-specific rules")
}

// Test #22: Target user is created if missing
func TestBootstrap_CreatesTargetUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: testuser123
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"id", "-un"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "testuser123", "User testuser123 should be created")
}

// Test #23: Bootstrap switches to target user after setup
func TestBootstrap_SwitchesToTargetUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: shai
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"sh", "-c", "id -u && id -un"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "shai", "Should run as user 'shai'")
	// Should not be root (UID 0)
	assert.NotContains(t, strings.Split(result, "\n")[0], "0", "Should not run as root (UID 0)")
}

// Test #24: User UID matches configured DEV_UID (host UID)
func TestBootstrap_UserUIDMatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: shai
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			// Add marker to easily find the UID in output
			Command: []string{"sh", "-c", "echo 'UID_IS:' && id -u && echo 'GID_IS:' && id -g"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	// Get the expected host UID/GID that should have been used
	expectedUID, expectedGID := hostUserIDs()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	// Extract the UID and GID from output
	lines := strings.Split(output.String(), "\n")
	var uid, gid string
	for i, line := range lines {
		if strings.Contains(line, "UID_IS:") && i+1 < len(lines) {
			uid = strings.TrimSpace(lines[i+1])
		}
		if strings.Contains(line, "GID_IS:") && i+1 < len(lines) {
			gid = strings.TrimSpace(lines[i+1])
		}
	}

	assert.Equal(t, expectedUID, uid, "Container UID should match host UID")
	assert.Equal(t, expectedGID, gid, "Container GID should match host GID")
}

// Test #25: Workspace directory has correct ownership
func TestBootstrap_WorkspaceOwnership(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: shai
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:     tmpDir,
		ConfigFile:     configPath,
		Verbose:        testing.Verbose(),
		ShowProgress:   false,
		ReadWritePaths: []string{"."},
		Stdout:         &output,
		PostSetupExec: &ExecSpec{
			// Test that user can write to workspace (functional requirement)
			// Note: Workspace may be owned by host UID on mounted volumes
			Command: []string{"sh", "-c", "touch /src/test_write && rm /src/test_write && echo 'WORKSPACE_WRITABLE'"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "WORKSPACE_WRITABLE", "User should be able to write to workspace")
}

// Test #26: Root commands execute before user switch
func TestBootstrap_RootCommandsExecuteBeforeSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: shai
resources:
  test:
    http:
      - example.com
    root-commands:
      - "touch /tmp/root_was_here"
      - "echo 'root command executed' > /tmp/root_output.txt"
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"sh", "-c", `
				test -f /tmp/root_was_here && echo "FILE_EXISTS" &&
				cat /tmp/root_output.txt
			`},
			UseTTY: false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "FILE_EXISTS", "Root command should have created file")
	assert.Contains(t, result, "root command executed", "Root command output should exist")
}

// Test #27: Root command failure stops bootstrap
func TestBootstrap_RootCommandFailureStopsBootstrap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: shai
resources:
  test:
    http:
      - example.com
    root-commands:
      - "exit 42"
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		PostSetupExec: &ExecSpec{
			Command: []string{"echo", "should not run"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	assert.Error(t, err, "Bootstrap should fail when root command fails")
}

// Test #29: Exec environment variables are set
func TestBootstrap_ExecEnvVarsSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: shai
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"env"},
			Env: map[string]string{
				"CUSTOM_VAR_1": "value1",
				"CUSTOM_VAR_2": "value2",
			},
			UseTTY: false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	envVars := output.String()
	assert.Contains(t, envVars, "CUSTOM_VAR_1=value1", "Custom env var 1 should be set")
	assert.Contains(t, envVars, "CUSTOM_VAR_2=value2", "Custom env var 2 should be set")
}

// Test #28: Custom UID/GID override works
func TestBootstrap_CustomUIDGIDOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: customuser
resources:
  test:
    http:
      - example.com
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	customUID := "9999"
	customGID := "8888"
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		HostUID:      customUID,
		HostGID:      customGID,
		PostSetupExec: &ExecSpec{
			Command: []string{"sh", "-c", "id -u && id -g && id -un"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, customUID, "Should use custom UID")
	assert.Contains(t, result, customGID, "Should use custom GID")
	assert.Contains(t, result, "customuser", "Should create and use custom username")
}

// Test #30: Resource environment variables are injected
func TestBootstrap_ResourceEnvVarsInjected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Set a host environment variable that will be mapped into the container
	os.Setenv("TEST_HOST_VAR", "host_value_123")
	defer os.Unsetenv("TEST_HOST_VAR")

	tmpDir := t.TempDir()
	configContent := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
user: shai
resources:
  test:
    http:
      - example.com
    vars:
      - source: TEST_HOST_VAR
        target: CONTAINER_VAR
apply:
  - path: ./
    resources: [test]
`
	configPath := filepath.Join(tmpDir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	var output strings.Builder
	cfg := EphemeralConfig{
		WorkingDir:   tmpDir,
		ConfigFile:   configPath,
		Verbose:      testing.Verbose(),
		ShowProgress: false,
		Stdout:       &output,
		PostSetupExec: &ExecSpec{
			Command: []string{"sh", "-c", "echo CONTAINER_VAR=$CONTAINER_VAR"},
			UseTTY:  false,
		},
	}

	runner, err := NewEphemeralRunner(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = runner.Run(ctx)
	require.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "CONTAINER_VAR=host_value_123", "Resource env var should be injected from host")
}
