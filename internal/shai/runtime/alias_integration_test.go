//go:build integration
// +build integration

package shai_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	shai "github.com/colony-2/shai/internal/shai/runtime"
	"github.com/stretchr/testify/require"
)

const testImage = "ghcr.io/colony-2/shai-base:latest"

func TestAliasIntegrationListShowsAliases(t *testing.T) {
	workspace := setupAliasWorkspace(t)
	listPayload := `{"jsonrpc":"2.0","id":1,"method":"listTools"}`
	cmd := fmt.Sprintf(`env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY curl --noproxy '*' -sS -H "Authorization: Bearer ${SHAI_ALIAS_TOKEN}" -H "Content-Type: application/json" -d '%s' "${SHAI_ALIAS_ENDPOINT}"`, listPayload)
	lines, err := runInSandbox(t, workspace, cmd)
	if err != nil {
		t.Fatalf("alias list failed: %v\nlogs: %v", err, lines)
	}
	assertContainsLine(t, lines, "hosthello")
	assertContainsLine(t, lines, "withargs")
	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)
}

func TestAliasIntegrationRunsHostCommand(t *testing.T) {
	workspace := setupAliasWorkspace(t)
	payload := `{"jsonrpc":"2.0","id":99,"method":"callTool","params":{"name":"hosthello","args":["first","second"]}}`
	cmd := fmt.Sprintf(`env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY curl --noproxy '*' -sS -H "Authorization: Bearer ${SHAI_ALIAS_TOKEN}" -H "Content-Type: application/json" -d '%s' "${SHAI_ALIAS_ENDPOINT}"`, payload)
	lines, err := runInSandbox(t, workspace, cmd)
	if err != nil {
		t.Fatalf("alias run failed: %v\nlogs: %v", err, lines)
	}
	assertContainsSubstring(t, lines, "HOST_HELLO:first second")
	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)
}

func TestAliasIntegrationArgValidation(t *testing.T) {
	workspace := setupAliasWorkspace(t)
	okPayload := `{"jsonrpc":"2.0","id":42,"method":"callTool","params":{"name":"withargs","args":["--msg=test"]}}`
	cmd := fmt.Sprintf(`env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY curl --noproxy '*' -sS -H "Authorization: Bearer ${SHAI_ALIAS_TOKEN}" -H "Content-Type: application/json" -d '%s' "${SHAI_ALIAS_ENDPOINT}"`, okPayload)
	lines, err := runInSandbox(t, workspace, cmd)
	if err != nil {
		t.Fatalf("alias run failed: %v\nlogs: %v", err, lines)
	}
	assertContainsSubstring(t, lines, "HOST_ARGS:--msg=test")
	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)

	badPayload := `{"jsonrpc":"2.0","id":43,"method":"callTool","params":{"name":"withargs","args":["--msg=Bad"]}}`
	badCmd := fmt.Sprintf(`env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY curl --noproxy '*' -sS -w "\n" -H "Authorization: Bearer ${SHAI_ALIAS_TOKEN}" -H "Content-Type: application/json" -d '%s' "${SHAI_ALIAS_ENDPOINT}"`, badPayload)
	lines, err = runInSandbox(t, workspace, badCmd)
	_ = err // validation failure expected
	assertContainsSubstring(t, lines, "arguments")
	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)
}

func TestAliasIntegrationShaiRemoteExecutesAlias(t *testing.T) {
	workspace := setupAliasWorkspace(t)
	cmd := `
set -euo pipefail
shai-remote list
shai-remote call hosthello foo bar
`
	lines, err := runInSandbox(t, workspace, cmd)
	if err != nil {
		t.Fatalf("shai-remote command failed: %v\nlogs: %v", err, lines)
	}
	assertContainsSubstring(t, lines, "hosthello")
	assertContainsSubstring(t, lines, "HOST_HELLO:foo bar")
	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)
}

func TestAliasIntegrationMasksManifest(t *testing.T) {
	workspace := setupAliasWorkspace(t)
	lines, err := runInSandbox(t, workspace, "if [ -e /src/.shai-cmds ]; then echo visible; else echo missing; fi")
	if err != nil {
		t.Fatalf("mask command failed: %v\nlogs: %v", err, lines)
	}
	assertContainsSubstring(t, lines, "missing")

	if _, err := os.Stat(filepath.Join(workspace, ".shai-cmds")); !os.IsNotExist(err) {
		t.Fatalf(".shai-cmds should not exist on host (err=%v)", err)
	}
	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)
}

// --- helpers ---

func setupAliasWorkspace(t *testing.T) string {
	t.Helper()
	requireDocker(t)

	workspace := t.TempDir()
	writeShaiConfig(t, workspace)

	scriptsDir := filepath.Join(workspace, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o777); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}

	writeExecutable(t, filepath.Join(scriptsDir, "host-hello.sh"), "#!/bin/sh\necho \"HOST_HELLO:$*\"\n")
	writeExecutable(t, filepath.Join(scriptsDir, "host-args.sh"), "#!/bin/sh\necho \"HOST_ARGS:$*\"\n")
	return workspace
}

func runInSandbox(t *testing.T, workspace, shellCmd string) ([]string, error) {
	t.Helper()
	t.Setenv("SHAI_ALIAS_DEBUG", "1")
	lines := newLineCollector()

	cfg := shai.EphemeralConfig{
		WorkingDir:     workspace,
		ReadWritePaths: []string{"."},
		Verbose:        true,
		PostSetupExec: &shai.ExecSpec{
			Command: []string{"/bin/bash", "-lc", shellCmd},
			Workdir: "/src",
			UseTTY:  false,
		},
		Stdout: lines.Writer("stdout"),
		Stderr: lines.Writer("stderr"),
	}
	runner, err := shai.NewEphemeralRunner(cfg)
	if err != nil {
		t.Fatalf("NewEphemeralRunner: %v", err)
	}
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := runner.Run(ctx); err != nil {
		return lines.Events(), err
	}
	return lines.Events(), nil
}

func requireDocker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docker info failed: %v\n%s", err, out)
	}

	cmd = exec.CommandContext(ctx, "docker", "image", "inspect", testImage)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docker image %s not available (build via `docker build -t %s docker`): %v\n%s", testImage, testImage, err, out)
	}
}

func writeShaiConfig(t *testing.T, workspace string) {
	t.Helper()
	cfgPath := filepath.Join(workspace, shai.DefaultConfigRelPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o777))
	config := fmt.Sprintf(`
type: shai-sandbox
version: 1
image: %s
resources:
  global:
    calls:
      - name: hosthello
        command: ./scripts/host-hello.sh
        allowed-args: .*
      - name: withargs
        command: ./scripts/host-args.sh
        allowed-args: ^(--msg=[a-z]+)$
apply:
  - path: ./
    resources: [global]
`, testImage)
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(config)+"\n"), 0o644); err != nil {
		t.Fatalf("write shai config: %v", err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

type lineCollector struct {
	lines []string
}

func newLineCollector() *lineCollector {
	return &lineCollector{lines: make([]string, 0, 32)}
}

func (l *lineCollector) Writer(stream string) io.Writer {
	return writerFunc(func(p []byte) (int, error) {
		chunks := strings.Split(string(p), "\n")
		for _, chunk := range chunks {
			trimmed := strings.TrimSpace(chunk)
			if trimmed == "" {
				continue
			}
			l.Collect(stream, trimmed)
		}
		return len(p), nil
	})
}

func (l *lineCollector) Collect(stream, line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	l.lines = append(l.lines, stream+":"+line)
	if stream == "stderr" {
		fmt.Fprintf(os.Stderr, "[alias-test stderr] %s\n", line)
	}
}

func (l *lineCollector) Events() []string {
	out := make([]string, len(l.lines))
	copy(out, l.lines)
	return out
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func assertContainsLine(t *testing.T, lines []string, needle string) {
	t.Helper()
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return
		}
	}
	logCollectedLines(t, lines)
	t.Fatalf("expected output to contain %q, got %v at %s", needle, lines, time.Now().Format(time.RFC3339))
}

func assertContainsSubstring(t *testing.T, lines []string, needle string) {
	t.Helper()
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return
		}
	}
	logCollectedLines(t, lines)
	t.Fatalf("expected substring %q in output %v at %s", needle, lines, time.Now().Format(time.RFC3339))
}

func logCollectedLines(t *testing.T, lines []string) {
	if len(lines) == 0 {
		t.Log("collected: <no output>")
		return
	}
	for _, line := range lines {
		t.Logf("collected: %s", line)
	}
}
