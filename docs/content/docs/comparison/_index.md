---
title: Comparison
weight: 11
---

How Shai compares to other tools and approaches.

## Shai vs Devcontainers

### Key Differences

| Feature | Shai | Devcontainers |
|---------|------|---------------|
| **Purpose** | Sandboxing AI agents | Development environments |
| **Lifecycle** | Ephemeral (minutes) | Long-lived (hours/days) |
| **Network Control** | Built-in filtering | Manual configuration |
| **Filesystem** | Read-only by default | Read-write by default |
| **Segmentation** | Cellular (per-component) | Monolithic (whole workspace) |
| **Setup Time** | Fast (<2s warm start) | Slower (features can take minutes) |
| **Configuration** | `.shai/config.yaml` | `.devcontainer/devcontainer.json` |
| **Apply Rules** | Path-based resource mapping | Not supported |
| **Remote Calls** | Built-in via MCP | Not supported |

### When to Use Devcontainers

- Long-lived development environments
- Complex IDE integration (VS Code)
- Team standardization on development tools
- Rich feature ecosystem

### When to Use Shai

- Running AI agents safely
- Cellular development workflows
- Ephemeral, task-specific environments
- Fine-grained network/filesystem controls
- Quick iteration cycles

### Can They Work Together?

Yes! Use devcontainers for your development environment and Shai for running AI agents:

```bash
# Inside your devcontainer
shai -rw src -- claude-code
```

## Shai vs Plain Docker

### What Shai Adds

| Feature | Plain Docker | Shai |
|---------|--------------|------|
| **Network Filtering** | Manual (complex iptables) | Automatic (HTTP allowlists) |
| **Cellular Access** | Manual mount configuration | Built-in (`-rw` paths) |
| **Configuration** | Dockerfile + docker-compose | `.shai/config.yaml` |
| **Resource Sets** | Not supported | Built-in |
| **Remote Calls** | Not supported | Built-in via MCP |
| **Bootstrap** | Manual scripts | Automatic |
| **User Management** | Manual | Automatic (UID/GID matching) |

### When to Use Plain Docker

- Building container images
- Production deployments
- Long-running services
- Maximum control and flexibility

### When to Use Shai

- Running AI agents
- Development sandboxing
- Quick prototyping
- Configuration-driven workflows

### Shai Uses Docker

Shai is built **on top of** Docker - it's not a replacement:

```
┌─────────────────────┐
│       Shai          │  ← High-level API
├─────────────────────┤
│      Docker         │  ← Container runtime
├─────────────────────┤
│   Linux Kernel      │
└─────────────────────┘
```

## Shai vs Virtual Machines

### Comparison

| Feature | VMs | Shai (Containers) |
|---------|-----|-------------------|
| **Isolation** | Strong (full kernel) | Good (namespace isolation) |
| **Startup Time** | Minutes | Seconds |
| **Resource Usage** | High (full OS) | Low (shared kernel) |
| **Disk Usage** | GBs per VM | MBs per container |
| **Network** | Virtual network required | Native Docker networking |
| **Filesystem** | Virtual disk | Direct bind mounts |

### When to Use VMs

- Strong isolation requirements (untrusted code)
- Different operating systems
- Kernel-level testing
- Legacy applications

### When to Use Shai

- Fast iteration cycles
- Resource efficiency
- Modern development workflows
- AI agent sandboxing

## Shai vs Firecracker/Kata

### Lightweight VMs

Firecracker and Kata Containers provide VM-level isolation with container-like speed.

| Feature | Shai | Firecracker/Kata |
|---------|------|------------------|
| **Isolation** | Container (namespaces) | VM (hypervisor) |
| **Startup** | ~1-2 seconds | ~1-2 seconds |
| **Memory Overhead** | ~50-100 MB | ~100-200 MB |
| **Use Case** | Development sandboxing | Production isolation |
| **Complexity** | Low | Higher |
| **Compatibility** | All Docker images | Special configuration |

### When to Use Firecracker/Kata

- Production multi-tenancy
- Untrusted code execution
- Strong isolation requirements
- Serverless platforms

### When to Use Shai

- Development workflows
- AI agent sandboxing
- Quick iteration
- Standard Docker images

## Shai vs nix/direnv

### Environment Management

| Feature | Shai | nix/direnv |
|---------|------|------------|
| **Approach** | Container-based | Shell-based |
| **Isolation** | Strong (container) | Weak (shell environment) |
| **Network Control** | Built-in | Not supported |
| **Filesystem** | Container filesystem | Host filesystem |
| **Setup** | Docker required | nix/direnv install |
| **Reproducibility** | High (images) | Very high (nix) |

### When to Use nix/direnv

- Pure development environments
- Reproducible builds
- No container overhead
- Shell-based workflows

### When to Use Shai

- AI agent sandboxing
- Network isolation needed
- Filesystem protection required
- Container-based workflows

## Summary

### Use Shai When

✅ Running AI agents
✅ Need network filtering
✅ Want read-only workspace by default
✅ Cellular development workflows
✅ Quick, ephemeral environments
✅ Fine-grained resource control

### Use Other Tools When

- **Devcontainers** - Long-lived dev environments, IDE integration
- **Plain Docker** - Building images, production deployments
- **VMs** - Strong isolation, different OSes
- **Firecracker/Kata** - Production multi-tenancy
- **nix/direnv** - Pure, reproducible shell environments

## Migration Guides

### From Devcontainers

```json
// .devcontainer/devcontainer.json
{
  "image": "mcr.microsoft.com/devcontainers/python:3.11",
  "features": {
    "ghcr.io/devcontainers/features/node:1": {}
  }
}
```

**Becomes:**

```yaml
# .shai/config.yaml
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-mega  # Includes Node + Python

resources:
  dev-tools:
    http:
      - npmjs.org
      - pypi.org

apply:
  - path: ./
    resources: [dev-tools]
```

### From Docker Compose

```yaml
# docker-compose.yml
services:
  app:
    image: node:20
    volumes:
      - .:/workspace
    environment:
      - API_KEY=${API_KEY}
```

**Becomes:**

```yaml
# .shai/config.yaml
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-mega

resources:
  app:
    vars:
      - source: API_KEY

apply:
  - path: ./
    resources: [app]
```

Usage:
```bash
# Instead of: docker-compose run app
shai -rw . -- npm start
```

## Best Practices

### Use the Right Tool

Don't force Shai where another tool fits better:

- **Development environment**: Devcontainers
- **Production**: Docker/Kubernetes
- **Build pipelines**: Plain Docker
- **AI agents**: Shai ✅

### Combine Tools

Use multiple tools together:

```bash
# Devcontainer for development
devcontainer up

# Shai for running agents inside devcontainer
shai -rw src -- claude-code
```

## Learn More

- [Core Concepts](/docs/concepts) - Understanding Shai's design
- [Quick Start](/docs/quick-start) - Get started with Shai
- [Examples](/docs/examples) - Real-world patterns
