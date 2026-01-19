---
title: Embedding Shai with Golang
weight: 7
---

Embed Shai inside Go applications for programmatic sandbox control.

## Installation

```bash
go get github.com/colony-2/shai/pkg/shai
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    shai "github.com/colony-2/shai/pkg/shai"
)

func main() {
    ctx := context.Background()

    // Load config from workspace
    cfg, err := shai.LoadSandboxConfig("/path/to/workspace",
        shai.WithReadWritePaths([]string{"src/components"}),
        shai.WithResourceSets([]string{"frontend-dev"}),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Set command to run
    cfg.PostSetupExec = &shai.SandboxExec{
        Command: []string{"npm", "test"},
        Workdir: "/src",
        UseTTY:  false,
    }

    // Create and run sandbox
    sandbox, err := shai.NewSandbox(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer sandbox.Close()

    if err := sandbox.Run(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Core Types

### SandboxConfig

Configuration for the sandbox:

```go
type SandboxConfig struct {
    WorkspacePath   string              // Path to workspace
    ConfigPath      string              // Path to .shai/config.yaml
    ReadWritePaths  []string            // Paths to mount as writable
    ResourceSets    []string            // Additional resource sets
    PrependResourceSet *ResourceSet     // Extra resources prepended to resolved sets
    AppendResourceSet  *ResourceSet     // Extra resources appended to resolved sets
    TemplateVars    map[string]string   // Template variables
    PostSetupExec   *SandboxExec        // Command to run
    ImageOverride   string              // Override image
    UserOverride    string              // Override user
    Verbose         bool                // Enable verbose output
    ShowScriptOutput bool               // Enable bootstrap warnings/banner
}
```

### SandboxExec

Command execution specification:

```go
type SandboxExec struct {
    Command []string          // Command and args
    Env     map[string]string // Additional env vars
    Workdir string            // Working directory
    UseTTY  bool              // Allocate TTY
}

### ResourceSet

Inline resources you can prepend/append from the Go API:

```go
type ResourceSet struct {
    Vars         []VarMapping
    Mounts       []Mount
    Calls        []Call
    HTTP         []string
    Ports        []Port
    RootCommands []string
    Options      ResourceOptions
}
```
```

### Sandbox Interface

```go
type Sandbox interface {
    Run(ctx context.Context) error        // Run and wait
    Start(ctx context.Context) (SandboxSession, error)  // Start without waiting
    Close() error                          // Cleanup
}
```

### SandboxSession

Long-running sandbox session:

```go
type SandboxSession interface {
    ContainerID() string               // Get container ID
    Wait() error                       // Wait for completion
    Stop(timeout time.Duration) error  // Stop gracefully
    Close() error                      // Cleanup
}
```

## Common Patterns

### Running Tests

```go
cfg, _ := shai.LoadSandboxConfig(workspacePath,
    shai.WithReadWritePaths([]string{"coverage"}),
)

cfg.PrependResourceSet = &shai.ResourceSet{
    HTTP: []string{"registry.npmjs.org"},
}

cfg.PostSetupExec = &shai.SandboxExec{
    Command: []string{"go", "test", "./..."},
    UseTTY:  false,
}

sandbox, _ := shai.NewSandbox(cfg)
defer sandbox.Close()

if err := sandbox.Run(ctx); err != nil {
    // Tests failed
}
```

### Long-Running Process

```go
sandbox, _ := shai.NewSandbox(cfg)
defer sandbox.Close()

// Start without waiting
session, err := sandbox.Start(ctx)
if err != nil {
    log.Fatal(err)
}
defer session.Close()

log.Printf("Container: %s", session.ContainerID())

// Do other work...

// Wait for completion
if err := session.Wait(); err != nil {
    log.Fatal(err)
}
```

### Multiple Sandboxes

```go
// Run multiple sandboxes concurrently
var wg sync.WaitGroup

for _, component := range components {
    wg.Add(1)
    go func(comp string) {
        defer wg.Done()

        cfg, _ := shai.LoadSandboxConfig(workspacePath,
            shai.WithReadWritePaths([]string{comp}),
        )

        sandbox, _ := shai.NewSandbox(cfg)
        defer sandbox.Close()

        sandbox.Run(ctx)
    }(component)
}

wg.Wait()
```

### With Template Variables

```go
cfg, _ := shai.LoadSandboxConfig(workspacePath,
    shai.WithTemplateVars(map[string]string{
        "ENV":    "staging",
        "REGION": "us-east-1",
    }),
)
```

## Error Handling

```go
cfg, err := shai.LoadSandboxConfig(workspacePath)
if err != nil {
    // Config loading failed
    // - Config file invalid
    // - Template vars missing
    // - Workspace doesn't exist
    log.Fatal(err)
}

sandbox, err := shai.NewSandbox(cfg)
if err != nil {
    // Sandbox creation failed
    // - Docker not available
    // - Image pull failed
    // - Invalid configuration
    log.Fatal(err)
}

if err := sandbox.Run(ctx); err != nil {
    // Execution failed
    // - Command failed
    // - Container crashed
    // - Network error
    log.Fatal(err)
}
```

## Testing with Shai

Use Shai in integration tests:

```go
func TestMyApp(t *testing.T) {
    ctx := context.Background()

    cfg, err := shai.LoadSandboxConfig(".",
        shai.WithReadWritePaths([]string{"tmp"}),
    )
    require.NoError(t, err)

    cfg.PostSetupExec = &shai.SandboxExec{
        Command: []string{"./test-script.sh"},
    }

    sandbox, err := shai.NewSandbox(cfg)
    require.NoError(t, err)
    defer sandbox.Close()

    err = sandbox.Run(ctx)
    assert.NoError(t, err)
}
```

## Advanced: Custom Logging

```go
import "os"

cfg.StdoutWriter = os.Stdout
cfg.StderrWriter = os.Stderr
cfg.Verbose = true

sandbox, _ := shai.NewSandbox(cfg)
// All output goes to configured writers
```

## See Also

- [GitHub Repository](https://github.com/colony-2/shai) for complete API docs
- [Examples](/docs/examples) for configuration patterns
- [pkg.go.dev](https://pkg.go.dev/github.com/colony-2/shai/pkg/shai) for API reference
