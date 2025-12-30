---
title: How Shai Works
weight: 5
---

Understanding Shai's architecture helps you use it effectively and troubleshoot issues.

## High-Level Overview

Shai creates a secure, ephemeral containerized environment for AI agents using Docker:

```
┌──────────────────────────────────────────────────────┐
│                    Your Machine                      │
│                                                      │
│  ┌────────────────────────────────────────────┐    │
│  │         Shai Container (Ephemeral)          │    │
│  │                                             │    │
│  │  ┌──────────────────────────────────┐      │    │
│  │  │     AI Agent (Claude, etc)       │      │    │
│  │  │     Running as non-root user     │      │    │
│  │  └──────────────────────────────────┘      │    │
│  │                                             │    │
│  │  Workspace: /src (read-only + overlays)    │    │
│  │  Network: Filtered (allowlist only)        │    │
│  │  Access: Controlled via resource sets      │    │
│  │                                             │    │
│  └────────────────────────────────────────────┘    │
│         ▲                              │            │
│         │ MCP                           │ iptables   │
│         │ (remote calls)                ▼ filtering │
│  ┌──────┴────────┐            ┌─────────────────┐  │
│  │  MCP Server   │            │  Host Network   │  │
│  └───────────────┘            └─────────────────┘  │
└──────────────────────────────────────────────────────┘
```

## Component Architecture

Shai consists of several components working together:

### 1. CLI (Command Line Interface)

The `shai` command you run from your terminal:
- Parses flags (`-rw`, `--resource-set`, etc.)
- Loads and validates `.shai/config.yaml`
- Resolves resource sets based on target paths
- Generates a bootstrap script
- Launches the Docker container

### 2. Docker Container

An ephemeral container with:
- **Name**: `shai-<random>` (e.g., `shai-a3f9b2`)
- **Auto-remove**: Deleted when you exit
- **Base image**: From config (e.g., `ghcr.io/colony-2/shai-mega`)
- **User**: Runs as non-root user (default: `shai`)

### 3. Bootstrap Script

A generated shell script that:
1. Creates the target user (`shai`)
2. Sets up network filtering (iptables, dnsmasq, tinyproxy)
3. Configures supervisord for background services
4. Mounts read-write overlays for specified paths
5. Logs firewall rules to `/var/log/shai/iptables.out`
6. Executes root commands (if specified)
7. Switches to the target user
8. Runs your command (or starts a shell)

### 4. Network Filtering

Three components enforce the HTTP/HTTPS allowlist:

**iptables**
- Drops all outbound traffic except:
  - DNS (port 53)
  - Proxy (tinyproxy)
  - Explicitly allowed ports
- Logs rules to `/var/log/shai/iptables.out`

**dnsmasq**
- DNS server that resolves only allowed domains
- Blocks DNS lookups for non-allowed domains
- Returns NXDOMAIN for blocked hosts

**tinyproxy**
- HTTP/HTTPS proxy
- Filters requests to allowed domains only
- Required for HTTPS (since iptables can't inspect encrypted traffic)

### 5. MCP Server

A host-side server that:
- Listens for remote call requests from the container
- Validates arguments against `allowed-args` regex
- Executes host commands
- Returns stdout/stderr to the container

Credentials for the MCP server are injected into the container as environment variables.

### 6. Filesystem Mounts

**Read-only workspace:**
```
Host: /path/to/your/project
Container: /src (read-only bind mount)
```

**Read-write overlays:**
```
Host: tmpfs
Container: /src/target/path (overlayfs)
```

For paths specified with `-rw`, Shai creates an overlayfs that:
- Uses the read-only workspace as the lower layer
- Uses a tmpfs (ephemeral memory) as the upper layer
- Presents a writable view to the agent

Changes are written to tmpfs and **discarded when the container stops**.

**Config protection:**

When the workspace root is mounted as read-write, Shai automatically remounts `.shai/config.yaml` as read-only to prevent sandbox escapes.

### 7. Supervisord

Background process manager that:
- Starts dnsmasq and tinyproxy
- Manages any additional services defined in `/etc/supervisor/conf.d/`
- Keeps services running if they crash

## Startup Flow

Here's what happens when you run `shai`:

```
1. Parse CLI flags
   ├─ Load .shai/config.yaml
   ├─ Validate configuration
   └─ Expand template variables

2. Resolve resource sets
   ├─ Find apply rules matching target paths
   ├─ Aggregate resource sets
   └─ Merge with --resource-set flags

3. Generate bootstrap script
   ├─ Create user setup commands
   ├─ Generate iptables rules
   ├─ Configure dnsmasq
   ├─ Configure tinyproxy
   ├─ Set up filesystem overlays
   └─ Add root commands

4. Start MCP server (if calls are defined)
   ├─ Generate credentials
   └─ Listen on random port

5. Launch Docker container
   ├─ Name: shai-<random>
   ├─ Image: from config
   ├─ Mounts: workspace + bootstrap script
   ├─ Auto-remove: true
   └─ Entrypoint: bootstrap script

6. Inside container:
   ├─ Run bootstrap script as root
   ├─ Set up user
   ├─ Configure network filtering
   ├─ Start supervisord
   ├─ Mount read-write overlays
   ├─ Execute root commands
   ├─ Switch to target user
   └─ Run your command (or start shell)

7. Agent runs in controlled environment

8. On exit:
   ├─ Stop MCP server
   ├─ Container auto-removes
   └─ Ephemeral changes discarded
```

## Security Layers

Shai provides multiple layers of security:

### Layer 1: Container Isolation

- Separate namespace from host
- Limited capabilities (no `CAP_SYS_ADMIN` by default)
- Resource limits (CPU, memory)
- No direct host access

### Layer 2: Filesystem Restrictions

- Workspace is read-only by default
- Write access only to specified paths
- Config file protected from modification
- Changes stored in ephemeral tmpfs

### Layer 3: Network Filtering

- iptables blocks non-allowed traffic
- DNS filtering prevents name resolution
- HTTP/HTTPS proxy enforces allowlist
- Explicit port allowances only

### Layer 4: User Isolation

- Runs as non-root user (`shai`)
- No sudo access
- Limited permissions
- Cannot modify system files

### Layer 5: Resource Control

- Only specified environment variables passed
- Only specified host directories mounted
- Only approved commands callable via MCP

### Layer 6: Ephemeral State

- Container auto-removes on exit
- No persistent changes to workspace
- Fresh environment each run
- No state leakage between sessions

## Resource Requirements

Shai requires these system utilities in the Docker image:

### Required Packages

- **supervisord** - Process supervisor
- **dnsmasq** - DNS server for domain filtering
- **iptables** - Firewall for network egress control
- **tinyproxy** - HTTP/HTTPS proxy for allow-listed traffic
- **bash** - Shell for bootstrap script
- **coreutils** - Basic Unix utilities
- **iproute2** - Network configuration
- **iputils-ping** - Network testing
- **jq** - JSON parsing
- **net-tools** - Network utilities
- **passwd** - User management
- **procps** - Process utilities
- **sed** - Text processing
- **util-linux** - System utilities

The `shai-base` and `shai-mega` images include all of these.

## Logging and Debugging

### Container Logs

View container logs:

```bash
docker logs shai-<random>
```

### Firewall Rules

Inside the container:

```bash
cat /var/log/shai/iptables.out
```

Shows the active iptables rules for network filtering.

### Verbose Mode

Run with `--verbose` to see:

```bash
shai -rw src --verbose
```

- Bootstrap script content
- Resource set resolution
- Network filtering configuration
- Mount points

### MCP Server Logs

MCP server logs are written to stderr of the `shai` process.

## Limitations

### What Shai Can't Do

- **Protect against malicious images**: If the Docker image itself is compromised, Shai can't help
- **Prevent resource exhaustion**: Agents can still use CPU/memory/disk (within Docker limits)
- **Guarantee agent correctness**: Shai provides sandboxing, not code validation
- **Work without Docker**: Shai requires Docker or a compatible container runtime

### Known Edge Cases

- **Docker-in-Docker**: Requires privileged mode and special configuration
- **GUI applications**: Not supported (Shai is CLI-focused)
- **Complex network protocols**: Only HTTP/HTTPS and explicit ports are supported
- **Large files**: Overlayfs writes to tmpfs, which is memory-backed

## Performance Considerations

### Startup Time

- **Cold start**: ~2-5 seconds (image pull, container creation, bootstrap)
- **Warm start**: ~1-2 seconds (image cached)

### Runtime Overhead

- **CPU**: Minimal (<5% overhead from proxy/filtering)
- **Memory**: ~100-200 MB for Shai infrastructure (dnsmasq, tinyproxy, supervisord)
- **Disk I/O**: Overlayfs adds minimal overhead for writes

### Optimization Tips

- Use `shai-base` for faster startup (smaller image)
- Keep resource sets focused (less network filtering rules)
- Avoid mounting large directories as read-write
- Pre-warm images: `docker pull ghcr.io/colony-2/shai-mega`

## Comparison with Other Tools

| Feature | Shai | Plain Docker | Devcontainers |
|---------|------|--------------|---------------|
| Network filtering | ✅ Built-in | ❌ Manual | ❌ Manual |
| Cellular development | ✅ Built-in | ❌ Manual | ❌ Not designed for this |
| Read-only workspace | ✅ Default | ❌ Manual | ❌ Typically writable |
| Ephemeral containers | ✅ Default | ⚠️ Manual | ❌ Long-lived |
| Remote calls | ✅ Built-in | ❌ N/A | ❌ N/A |
| Config-driven | ✅ Yes | ⚠️ Dockerfile only | ✅ Yes |
| AI agent focus | ✅ Yes | ❌ No | ❌ No |

## Next Steps

- Understand [Security](/docs/security) in depth
- See [Docker Images](/docs/docker-images) for image requirements
- Browse [Troubleshooting](/docs/troubleshooting) for common issues
