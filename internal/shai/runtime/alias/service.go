package alias

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/colony-2/shai/internal/shai/runtime/alias/mcp"
)

const (
	defaultShell       = "/bin/bash"
	defaultExecTimeout = 10 * time.Minute
)

// Config contains inputs required to start the alias subsystem.
type Config struct {
	WorkingDir     string
	ShellPath      string
	Debug          bool
	Entries        []*Entry
	DockerHostAddr string
	MCPBindAddr    string
}

// Service manages the lifecycle of the alias MCP server.
type Service struct {
	env            []string
	server         *mcp.Server
	closeOnce      sync.Once
	dockerHostAddr string
}

// MaybeStart initializes the alias system, even if no entries are supplied.
func MaybeStart(cfg Config) (*Service, error) {
	var entries []*Entry
	for _, entry := range cfg.Entries {
		if entry != nil {
			entries = append(entries, entry)
		}
	}
	workingDir := cfg.WorkingDir
	if strings.TrimSpace(workingDir) == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve working directory: %w", err)
		}
		workingDir = dir
	} else {
		if !filepath.IsAbs(workingDir) {
			abs, err := filepath.Abs(workingDir)
			if err != nil {
				return nil, fmt.Errorf("resolve working directory: %w", err)
			}
			workingDir = abs
		}
	}

	shellPath := cfg.ShellPath
	if strings.TrimSpace(shellPath) == "" {
		shellPath = os.Getenv("SHELL")
	}
	if strings.TrimSpace(shellPath) == "" {
		shellPath = defaultShell
	}

	dockerHostAddr := cfg.DockerHostAddr
	if strings.TrimSpace(dockerHostAddr) == "" {
		dockerHostAddr = "host.docker.internal"
	}

	mcpBindAddr := cfg.MCPBindAddr
	if strings.TrimSpace(mcpBindAddr) == "" {
		mcpBindAddr = "0.0.0.0:0"
	}

	token, err := randomBase64(32)
	if err != nil {
		return nil, fmt.Errorf("generate alias token: %w", err)
	}
	sessionID, err := randomBase64(16)
	if err != nil {
		return nil, fmt.Errorf("generate session id: %w", err)
	}

	executor := &Executor{
		WorkingDir: workingDir,
		ShellPath:  shellPath,
		Timeout:    defaultExecTimeout,
	}

	server, err := mcp.NewServer(mcp.Config{
		BindAddr:      mcpBindAddr,
		Token:         token,
		SessionID:     sessionID,
		Executor:      newAliasExecutorAdapter(executor, entries),
		MaxConcurrent: 4,
	})
	if err != nil {
		return nil, fmt.Errorf("start alias MCP server: %w", err)
	}
	server.Start()

	port := server.Port()
	envList := []string{
		fmt.Sprintf("SHAI_ALIAS_ENDPOINT=http://%s:%d/mcp", dockerHostAddr, port),
		fmt.Sprintf("SHAI_ALIAS_TOKEN=%s", token),
		fmt.Sprintf("SHAI_ALIAS_SESSION_ID=%s", sessionID),
		fmt.Sprintf("ALLOW_DOCKER_HOST_PORT=%d", port),
	}

	return &Service{
		env:            envList,
		server:         server,
		dockerHostAddr: dockerHostAddr,
	}, nil
}

// Env returns environment variables to inject into the container.
func (s *Service) Env() []string {
	out := make([]string, len(s.env))
	copy(out, s.env)
	return out
}

// Close terminates the MCP server.
func (s *Service) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.server != nil {
			_ = s.server.Close(context.Background())
		}
	})
}

func randomBase64(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

type aliasExecutorAdapter struct {
	exec    *Executor
	entries map[string]*Entry
	tools   []mcp.Tool
}

func newAliasExecutorAdapter(exec *Executor, entries []*Entry) *aliasExecutorAdapter {
	entryMap := make(map[string]*Entry, len(entries))
	tools := make([]mcp.Tool, 0, len(entries))
	for _, e := range entries {
		if e == nil || e.Name == "" {
			continue
		}
		entryMap[e.Name] = e
		tools = append(tools, mcp.Tool{
			Name:        e.Name,
			Description: e.Description,
		})
	}
	return &aliasExecutorAdapter{
		exec:    exec,
		entries: entryMap,
		tools:   tools,
	}
}

func (a *aliasExecutorAdapter) Tools() []mcp.Tool {
	out := make([]mcp.Tool, len(a.tools))
	copy(out, a.tools)
	return out
}

func (a *aliasExecutorAdapter) Execute(ctx context.Context, name string, args []string, streams mcp.Streams) (int, error) {
	entry, ok := a.entries[name]
	if !ok {
		return 0, fmt.Errorf("alias %q not found", name)
	}
	result, err := a.exec.Run(ctx, entry, args, Streams{
		Stdout: streams.Stdout,
		Stderr: streams.Stderr,
	})
	if err != nil {
		return 0, err
	}
	return result.ExitCode, nil
}
