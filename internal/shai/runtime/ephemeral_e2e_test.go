package shai_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	shai "github.com/colony-2/shai/internal/shai/runtime"
)

// This test exercises the full EphemeralRunner path against a real Docker daemon.
func TestEphemeralRunner_E2ECommonUtils(t *testing.T) {
	t.Parallel()
	requireDockerAvailable(t)

	tmpDir := t.TempDir()

	cfg := `
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
	cfgPath := filepath.Join(tmpDir, shai.DefaultConfigRelPath)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o777); err != nil {
		t.Fatalf("mkdir shai config dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)+"\n"), 0o644); err != nil {
		t.Fatalf("write shai config: %v", err)
	}

	runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
		WorkingDir:     tmpDir,
		ReadWritePaths: []string{"."},
		PostSetupExec: &shai.ExecSpec{
			Command: []string{"/bin/bash", "-lc", "echo 'e2e ok'"},
			UseTTY:  false,
		},
	})
	if err != nil {
		t.Fatalf("runner create: %v", err)
	}
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("ephemeral run failed: %v", err)
	}

	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)

}

func TestEphemeralRunner_ProxyStack(t *testing.T) {
	t.Parallel()
	requireDockerAvailable(t)

	tmpDir := t.TempDir()

	cfg := `
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-base:latest
resources:
  base:
    http:
      - example.com
apply:
  - path: ./
    resources:
      - base
`
	cfgPath := filepath.Join(tmpDir, shai.DefaultConfigRelPath)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o777); err != nil {
		t.Fatalf("mkdir shai config dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)+"\n"), 0o644); err != nil {
		t.Fatalf("write shai config: %v", err)
	}

	script := `
set -euo pipefail
sleep 3
require_proc() {
  local name=$1
  if command -v pgrep >/dev/null 2>&1; then
    pgrep -x "$name" >/dev/null 2>&1 && return 0
    pgrep -f "$name" >/dev/null 2>&1 && return 0
  elif command -v pidof >/dev/null 2>&1; then
    pidof "$name" >/dev/null 2>&1 && return 0
  fi
  if ps -eo comm >/dev/null 2>&1; then
    if ps -eo comm | grep -Fx "$name" >/dev/null 2>&1; then
      return 0
    fi
  fi
  echo "process $name not found" >&2
  return 1
}

require_proc tinyproxy
require_proc dnsmasq

curl -sSfk --connect-timeout 10 --max-time 20 https://example.com >/tmp/allowed.html
if [ ! -s /tmp/allowed.html ]; then
  echo "no data fetched from allowed domain" >&2
  exit 1
fi

set +e
curl -sSfk --connect-timeout 5 --max-time 10 https://example.net >/tmp/deny.html
status=$?
set -e
if [ "$status" -eq 0 ]; then
  echo "disallowed domain unexpectedly succeeded" >&2
  exit 1
fi
`

	runner, err := shai.NewEphemeralRunner(shai.EphemeralConfig{
		WorkingDir:     tmpDir,
		ReadWritePaths: []string{"."},
		PostSetupExec: &shai.ExecSpec{
			Command: []string{"/bin/bash", "-lc", script},
			UseTTY:  false,
		},
	})
	if err != nil {
		t.Fatalf("runner create: %v", err)
	}
	defer runner.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("proxy stack validation failed: %v", err)
	}

	// wait second or container rm will fail due to macos conccurrency issues with virtiofs
	time.Sleep(1 * time.Second)
}
