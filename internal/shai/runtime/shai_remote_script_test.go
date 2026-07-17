package shai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type rpcRequest struct {
	Method string          `json:"method"`
	ID     json.RawMessage `json:"id"`
	Params json.RawMessage `json:"params"`
}

type callToolParams struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

func TestShaiRemoteList(t *testing.T) {
	srv := newAliasServer(t, "test-token", func(t *testing.T, body []byte) []byte {
		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Method != "listTools" {
			t.Fatalf("expected listTools method, got %q", req.Method)
		}
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"tools": []map[string]string{
					{"name": "hosthello", "description": "echo hello"},
					{"name": "withdesc"},
				},
			},
		}
		out, _ := json.Marshal(resp)
		return out
	})
	defer srv.Close()

	stdout, stderr, code := runShaiRemote(t, []string{
		"SHAI_ALIAS_ENDPOINT=" + srv.URL,
		"SHAI_ALIAS_TOKEN=test-token",
		"SHAI_ALIAS_SESSION_ID=session-1",
	}, "list")

	if code != 0 {
		t.Fatalf("list exited with %d stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 calls, got %q", stdout)
	}
	if lines[0] != "hosthello - echo hello" {
		t.Fatalf("unexpected first line: %q", lines[0])
	}
	if lines[1] != "withdesc" {
		t.Fatalf("unexpected second line: %q", lines[1])
	}
}

func TestShaiRemoteExecStreamsOutput(t *testing.T) {
	srv := newAliasServer(t, "test-token", func(t *testing.T, body []byte) []byte {
		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Method != "callTool" {
			t.Fatalf("unexpected method %q", req.Method)
		}
		var params callToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			t.Fatalf("decode params: %v", err)
		}
		if params.Name != "hosthello" {
			t.Fatalf("expected call hosthello, got %q", params.Name)
		}
		expectedArgs := []string{"first", "--flag=value"}
		if got := params.Args; len(got) != len(expectedArgs) {
			t.Fatalf("expected %d args, got %v", len(expectedArgs), got)
		} else {
			for i, arg := range expectedArgs {
				if got[i] != arg {
					t.Fatalf("arg %d mismatch: want %q got %q", i, arg, got[i])
				}
			}
		}
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"content": []map[string]string{
					{"type": "text", "role": "stdout", "text": "hello world\n"},
					{"type": "text", "role": "stderr", "text": "warn\n"},
				},
				"exitCode": 7,
			},
		}
		out, _ := json.Marshal(resp)
		return out
	})
	defer srv.Close()

	stdout, stderr, code := runShaiRemote(t, []string{
		"SHAI_ALIAS_ENDPOINT=" + srv.URL,
		"SHAI_ALIAS_TOKEN=test-token",
	}, "call", "hosthello", "first", "--flag=value")

	if code != 7 {
		t.Fatalf("expected exit code 7, got %d stderr=%q stdout=%q", code, stderr, stdout)
	}
	if stdout != "hello world\n" {
		t.Fatalf("unexpected stdout %q", stdout)
	}
	if stderr != "warn\n" {
		t.Fatalf("unexpected stderr %q", stderr)
	}
}

func TestShaiRemoteExecError(t *testing.T) {
	const overrideToken = "override-token"
	srv := newAliasServer(t, overrideToken, func(t *testing.T, body []byte) []byte {
		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error": map[string]string{
				"message": "call not found",
			},
		}
		out, _ := json.Marshal(resp)
		return out
	})
	defer srv.Close()

	stdout, stderr, code := runShaiRemote(t, []string{
		"SHAI_ALIAS_ENDPOINT=wrong",
		"SHAI_ALIAS_TOKEN=wrong",
	}, "call", "--endpoint", srv.URL, "--token", overrideToken, "missing")

	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if stdout != "" {
		t.Fatalf("expected no stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "call not found") {
		t.Fatalf("stderr %q missing error message", stderr)
	}
}

func newAliasServer(t *testing.T, expectedToken string, responder func(t *testing.T, body []byte) []byte) *httptest.Server {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if got := r.Header.Get("Authorization"); got != "Bearer "+expectedToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		resp := responder(t, body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})
	srv := httptest.NewServer(handler)
	return srv
}

func runShaiRemote(t *testing.T, extraEnv []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := scriptPath(t)

	// On Windows, we need to invoke bash explicitly since the shell script won't execute directly
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmdArgs := append([]string{script}, args...)
		cmd = exec.CommandContext(ctx, "bash", cmdArgs...)
	} else {
		cmd = exec.CommandContext(ctx, script, args...)
	}

	cmd.Env = append(os.Environ(), extraEnv...)
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err == nil {
		return stdout, stderr, 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdout, stderr, exitErr.ExitCode()
	}
	t.Fatalf("command failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	return "", "", 0
}

func scriptPath(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)

	// shai-remote is a shell script that we invoke directly on Unix
	// or via bash on Windows (see runShaiRemote)
	script := filepath.Join(root, "internal", "shai", "runtime", "bootstrap", "shai-remote.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("shai-remote script missing: %v", err)
	}
	return script
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("unable to locate go.mod from %s", dir)
		}
		dir = parent
	}
}
