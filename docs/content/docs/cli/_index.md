---
title: CLI Reference
weight: 8
---

Complete command-line interface reference for Shai.

## Commands

### `shai`

Main command - starts a sandbox in the current directory.

```bash
shai [flags] [-- command [args...]]
```

### `shai generate`

Generate a default `.shai/config.yaml` file.

```bash
shai generate
```

Creates `.shai/config.yaml` with sensible defaults.

### `shai version`

Display version information.

```bash
shai version
```

## Flags

### `--read-write, -rw <path>`

**Repeatable:** Yes

Mark a path as writable. Path must exist.

```bash
shai -rw src/components
shai -rw src -rw tests
```

### `--resource-set, -rs <name>`

**Repeatable:** Yes

Opt into additional resource sets beyond config's apply rules.

```bash
shai -rw src --resource-set gpu-access
shai -rs tools-1 -rs tools-2
```

### `--image, -i <image>`

Override the container image.

```bash
shai --image ghcr.io/my-org/custom:latest
```

Takes precedence over config file and apply rules.

### `--user, -u <user>`

Override the target container user.

```bash
shai --user developer
```

Default: `shai`

### `--privileged`

Run the container in privileged mode.

```bash
shai --privileged
```

{{< callout type="warning" >}}
Reduces container isolation. Use with caution.
{{< /callout >}}

### `--var, -v KEY=value`

**Repeatable:** Yes

Provide template variables for config.

```bash
shai --var TAG=v1.2.3 --var ENV=staging
```

Used with `${{ vars.KEY }}` in config.

### `--verbose, -V`

Dump bootstrap details and resource decisions.

```bash
shai --verbose
```

Shows:
- Bootstrap script
- Resolved resource sets
- Network filtering rules
- Mount points

### `--no-tty, -T`

Disable TTY allocation.

```bash
shai --no-tty -- npm test
```

Useful for structured log output and CI/CD.

### `--config <path>`

Override config file location.

```bash
shai --config /path/to/custom-config.yaml
```

Default: `.shai/config.yaml` in workspace root

### `--help, -h`

Show help message.

```bash
shai --help
```

## Post-Setup Command

Everything after `--` is executed inside the sandbox after setup:

```bash
shai -rw src -- npm test
shai -rw backend -- go build ./...
shai -rw . -- bash -c "echo hello && pwd"
```

Without `--`, Shai drops you into an interactive shell.

## Examples

### Basic Usage

```bash
# Interactive shell (read-only)
shai

# Interactive shell (src writable)
shai -rw src

# Run command
shai -- npm test

# Run command with write access
shai -rw src -- npm run build
```

### Multiple Paths

```bash
# Multiple writable paths
shai -rw src/auth -rw src/payments -rw tests

# Run tests with write access to coverage output
shai -rw coverage -- npm test
```

### Resource Sets

```bash
# Add resource sets
shai -rw ml --resource-set gpu-access --resource-set wandb-api

# Override image
shai -rw src --image ghcr.io/my-org/dev:latest

# Custom user
shai -rw backend --user developer
```

### Template Variables

```bash
# Provide template vars
shai --var ENV=production --var REGION=us-west-2 -rw infrastructure

# Multiple vars
shai --var TAG=v2.0.0 --var DEBUG=true
```

### Debugging

```bash
# Verbose output
shai -rw src --verbose

# Inspect bootstrap without running
shai -rw src --verbose -- /bin/true

# No TTY for logs
shai --no-tty -- ./run-tests.sh > test.log
```

### Complex Commands

```bash
# Chain commands
shai -- bash -c "npm install && npm test"

# With environment
shai -- env NODE_ENV=test npm test

# Multiple commands
shai -- sh -c "echo start && npm run build && echo done"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Docker not available |
| 3 | Config invalid |
| 4 | Path doesn't exist |
| 125 | Docker daemon error |
| 126 | Command cannot execute |
| 127 | Command not found |
| 130 | Terminated by Ctrl+C |

## Environment Variables

### `DOCKER_HOST`

Override Docker daemon location.

```bash
export DOCKER_HOST=unix:///var/run/docker.sock
shai
```

### `SHAI_CONFIG`

Default config file location.

```bash
export SHAI_CONFIG=/path/to/config.yaml
shai
```

## Shell Completion

### Bash

```bash
shai completion bash > /etc/bash_completion.d/shai
```

### Zsh

```bash
shai completion zsh > ~/.zsh/completions/_shai
```

### Fish

```bash
shai completion fish > ~/.config/fish/completions/shai.fish
```

## Tips

### Quick Iteration

```bash
# Use short flags
shai -rw src -V -- npm test

# Omit redundant flags
shai -rw .  # Entire workspace writable
```

### Combining Flags

```bash
# All flags together
shai \
  --image custom:latest \
  --user dev \
  --resource-set tools \
  --var TAG=v1.0.0 \
  --verbose \
  -rw src \
  -- npm run build
```

### Default Behavior

```bash
# No command = interactive shell
shai -rw src

# No -rw = read-only workspace
shai

# No config = uses embedded defaults
shai  # Works even without .shai/config.yaml
```

## See Also

- [Quick Start](/docs/quick-start) for getting started
- [Configuration Reference](/docs/configuration) for config options
- [Examples](/docs/examples) for common patterns
