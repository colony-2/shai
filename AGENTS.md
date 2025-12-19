# server/container

## Overview

Sandbox orchestration system for vibethis "boxes". It provisions ephemeral Docker containers using `.shai/config.yaml`, applies selective read-write overlays, and exposes both a CLI (`cmd/shai`) and a reusable API (`pkg/shai`) for launching workloads with strict workspace isolation.

## Architecture

### Container Internals (`internal/container/`)
- **Manager Interface**: Public contract kept for historical compatibility; production paths go through the sandbox API instead.
- **Info / Status / Mount**: Shared type definitions for callers that still reason about docker-like lifecycle states.

### Sandbox Runtime (`internal/shai/runtime/`)
- **EphemeralRunner**: Core engine that loads `.shai/config.yaml`, resolves resource sets, builds Docker options, and supervises containers (`ephemeral_runner.go`).
- **Config Loader (`internal/shai/runtime/config/`)**: Parses YAML, performs template expansion, validates resource/call definitions, and resolves apply rules.
- **MountBuilder**: Implements selective read-write overlays (base repo at `/src` read-only; provided paths become writable, with optional whole-workspace mode guarded by `.shai` dir protection).
- **Alias + Bootstrap**: Auxiliary services that expose host side commands inside the sandbox and generate the setup script executed before user processes run.
- **Docker Integration**: Lazy client detection (Desktop sockets on macOS, env overrides, rootless paths), image validation/pulls, and auto-removal semantics when containers exit.

### Shai CLI (`pkg/shai/` & `cmd/shai/`)
- **Sandbox API**: `pkg/shai.Sandbox` exposes `Run`, `Start`, and `Close`, delegating to the internal runtime.
- **SandboxConfig**: Public struct accepted by both CLI and library consumers (working dir, `.shai` config path, read-write overlays, resource sets, image overrides, verbosity, writers, exec specification).
- **CLI Wiring**: Cobra-based command that normalizes legacy flags, optionally collects read-write paths, captures template variables, and forwards everything to the sandbox runtime.

## Programmatic Usage

Shai exposes a generic API to run any process inside an ephemeral sandbox after loading `.shai/config.yaml`. Consumers provide a working directory, optional overlays, and (optionally) a `SandboxExec` describing the command to run post-setup.

### Key Types (pkg/shai)
```go
type SandboxExec struct {
    Command []string          // argv to exec after setup
    Env     map[string]string // extra env vars
    Workdir string            // container cwd; default: workspaceFolder from config
    UseTTY  bool              // default true; false => demuxed logs
}

type SandboxConfig struct {
    WorkingDir          string
    ConfigFile          string
    TemplateVars        map[string]string
    ReadWritePaths      []string
    ResourceSets        []string
    Verbose             bool
    PostSetupExec       *SandboxExec
    Stdout              io.Writer
    Stderr              io.Writer
    GracefulStopTimeout time.Duration
    ImageOverride       string
}

type Sandbox interface {
    Run(ctx context.Context) error
    Start(ctx context.Context) (*SandboxSession, error)
    Close() error
}

type SandboxSession struct {
    ContainerID string
    Wait(ctx context.Context) error
    Stop(ctx context.Context) error
    Close() error
}
```

### Example: Run a custom entrypoint
```go
import (
  "context"
  shai "github.com/colony-2/shai/pkg/shai"
)

func startProcess(ctx context.Context, repoRoot string, logs io.Writer) (*shai.SandboxSession, error) {
  cfg := shai.SandboxConfig{
    WorkingDir:     repoRoot,
    ReadWritePaths: []string{".vibethis", ".cache"},
    Stdout:         logs,
    Stderr:         logs,
    PostSetupExec: &shai.SandboxExec{
      Command: []string{"./bin/worker", "--name", "example"},
      UseTTY: false, // structured logs
    },
  }

  sandbox, err := shai.NewSandbox(cfg)
  if err != nil { return nil, err }
  sess, err := sandbox.Start(ctx)
  if err != nil { _ = sandbox.Close(); return nil, err }
  return sess, nil
}
```

## Behavior
- `.shai/config.yaml` fully defines image, workspace path, container user, resource sets, host-call shims, HTTP allow lists, and optional image overrides via path-based apply rules.
- Devcontainer lifecycle hooks are no longer processed. Instead, Shai emits a bootstrap script that wires environment variables, resolves host-call aliases, applies resource mounts, and then hands control to the requested command (or login shell).
- If `PostSetupExec` is nil: the runner switches to the configured user and launches an interactive login shell attached to the caller’s TTY.
- If `PostSetupExec` is set: the runner exports the provided environment, optionally `cd`s, and executes the supplied command with or without a TTY.
- Mounting: `/src` is read-only by default; entries in `ReadWritePaths` become writable bind mounts. Passing `.` switches the entire workspace to RW mode and then immediately remounts `.shai/` as read-only to guard configs.
- `--resource-set/-rs` flags activate named resource sets from the config file (CLI-specified sets append to those inferred from apply rules).
- `--image/-i` overrides the container image explicitly and wins over apply-rule overrides.
- Verbose mode dumps the generated setup script, resource selection details, and per-phase progress markers from the bootstrap process.

## Configuration Quick Reference
- Location: `${repo}/.shai/config.yaml` by default; `--config` flag overrides it.
- Schema essentials:
  - `type: shai-sandbox`, `version: 1`
  - `image`, `user`, `workspace`
  - `resources`: named bundles containing `calls`, `mounts`, `vars`, `http`, `ports`
  - `apply`: ordered list mapping relative paths to resource sets and optional image overrides (paths resolved against workspace; `"./"` cannot change the image).
- Template expansion: supports `${{ env.NAME }}` (host environment) and `${{ vars.NAME }}` (CLI `--var`).
- Validation ensures referenced resource sets exist, call names are unique, allowed-args regexes compile, and mounts declare valid modes (`ro`/`rw`).

## Core Runtime Flow
1. Load `.shai/config.yaml` (or default template) and expand templates with host env + CLI vars.
2. Resolve resource sets (apply rules + CLI overrides) and collect required mounts, host-call scripts, env injections, and HTTP allow lists.
3. Initialize `MountBuilder` to construct Docker bind mounts with selective RW overlays.
4. Materialize the alias manifest and bootstrap scripts under `.shai/`.
5. Create/start a Docker container with the resolved image, workspace, user, mounts, env, and auto-remove flag.
6. Run bootstrap -> optional `PostSetupExec` -> interactive shell/command.
7. Stream logs/TTY according to `UseTTY`; expose `SandboxSession` for long-running processes.

## Usage Examples
```bash
# Minimal invocation (read-only workspace)
shai

# Mark specific directories as writable
shai --read-write src

# Multiple read-write paths
shai --read-write src --read-write tests --read-write docs

# Entire workspace read-write plus template vars
shai -rw . --var BRANCH=feature --resource-set staging-secrets

# Verbose logging with explicit image override
shai --read-write src --verbose --image ghcr.io/example/dev:latest
```

## Progress Display
```
⠋ Checking image availability
✅ Image found locally
⠋ Creating container
✅ Container created
⠋ Running sandbox bootstrap
✅ Ready for commands
```
