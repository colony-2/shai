---
title: Comparison & Prior Art
weight: 11
---

Understanding where Shai came from and how it compares to other tools.

## Inspiration & Origins

### Built on Devcontainers

The first version of Shai was built on top of [devcontainers](https://containers.dev). The original underlying library can be [seen here](https://github.com/colony-2/devcontainer-go).

Devcontainers are great and were a good place for Shai to start. However, over time, the design goal differences between Shai and Devcontainers became challenging, and we ultimately decided to define an alternative configuration approach.

### Why We Moved Away

While devcontainers excel at their intended use case, several fundamental differences made them unsuitable as Shai's foundation:

**Configuration Segmentation** - The devcontainer spec is not designed for segmented configuration. There's no native way to specify that subdirectory `a` should get different resources than subdirectory `b`. Shai's cellular development model requires this kind of path-based resource mapping.

**Lifecycle Expectations** - Devcontainers are expected to be longer-lived development environments. Features can take 30 seconds to many minutes to install. Using features in Shai meant every new session had a large startup wait, which conflicts with Shai's goal of ephemeral, task-specific containers that start in seconds.

**Feature Scope** - There are many features in devcontainers. Shai only needed a small subset of them. The additional complexity wasn't providing value for Shai's specific use case of sandboxing AI agents.

**Sandboxing Controls** - Devcontainers don't have built-in sandboxing controls for things like network firewalling or filesystem restrictions. You can define these in a feature or container image, but they're basically DIY. Shai needed these controls to be first-class primitives.

**Ephemeral vs Persistent** - The devcontainers tools are not really built for throw-away ephemeral containers. They assume some level of persistence and state management that doesn't align with Shai's model of starting fresh for each task.


### Design Philosophy

The move away from devcontainers led to several core Shai design principles:

1. **Fast startup** - No features, no long initialization
2. **Sandboxing first** - Network filtering and filesystem controls are built-in
3. **Cellular by default** - Path-based resource sets enable fine-grained access control
4. **Ephemeral** - Assume containers are disposable
5. **Configuration safety** - Actively prevent credential leakage
6. **Minimal scope** - Only include what's needed for AI agent sandboxing

**Shai has no desire to replace devcontainers.** They are focused on two different use cases.

## Comparing Shai to Other Tools

### Shai vs Devcontainers

#### Key Differences

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

#### When to Use Each

**Use Devcontainers for:**
- Long-lived development environments
- Complex IDE integration (VS Code)
- Team standardization on development tools
- Rich feature ecosystem

**Use Shai for:**
- Running AI agents safely
- Cellular development workflows
- Ephemeral, task-specific environments
- Fine-grained network/filesystem controls
- Quick iteration cycles

#### Can They Work Together?

Yes! Use devcontainers for your development environment and Shai for running AI agents:

```bash
# Inside your devcontainer
shai -rw src -- claude
```

### Shai vs Plain Docker

#### What Shai Adds

| Feature | Plain Docker | Shai |
|---------|--------------|------|
| **Network Filtering** | Manual (complex iptables) | Automatic (HTTP allowlists) |
| **Cellular Access** | Manual mount configuration | Built-in (`-rw` paths) |
| **Configuration** | Dockerfile + docker-compose | `.shai/config.yaml` |
| **Resource Sets** | Not supported | Built-in |
| **Remote Calls** | Not supported | Built-in via MCP |
| **Bootstrap** | Manual scripts | Automatic |
| **User Management** | Manual | Automatic (UID/GID matching) |

#### When to Use Each

**Use Plain Docker for:**
- Building container images
- Production deployments
- Long-running services
- Maximum control and flexibility

**Use Shai for:**
- Running AI agents
- Development sandboxing
- Quick prototyping
- Configuration-driven workflows

#### Shai Uses Docker

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

### Shai vs Virtual Machines

#### Comparison

| Feature | VMs | Shai (Containers) |
|---------|-----|-------------------|
| **Isolation** | Strong (full kernel) | Good (namespace isolation) |
| **Startup Time** | Minutes | Seconds |
| **Resource Usage** | High (full OS) | Low (shared kernel) |
| **Disk Usage** | GBs per VM | MBs per container |
| **Network** | Virtual network required | Native Docker networking |
| **Filesystem** | Virtual disk | Direct bind mounts |

#### When to Use Each

**Use VMs for:**
- Strong isolation requirements (untrusted code)
- Different operating systems
- Kernel-level testing
- Legacy applications

**Use Shai for:**
- Fast iteration cycles
- Resource efficiency
- Modern development workflows
- AI agent sandboxing

### Shai vs Firecracker/Kata

#### Lightweight VMs

Firecracker and Kata Containers provide VM-level isolation with container-like speed.

| Feature | Shai | Firecracker/Kata |
|---------|------|------------------|
| **Isolation** | Container (namespaces) | VM (hypervisor) |
| **Startup** | ~1-2 seconds | ~1-2 seconds |
| **Memory Overhead** | ~50-100 MB | ~100-200 MB |
| **Use Case** | Development sandboxing | Production isolation |
| **Complexity** | Low | Higher |
| **Compatibility** | All Docker images | Special configuration |

#### When to Use Each

**Use Firecracker/Kata for:**
- Production multi-tenancy
- Untrusted code execution
- Strong isolation requirements
- Serverless platforms

**Use Shai for:**
- Development workflows
- AI agent sandboxing
- Quick iteration
- Standard Docker images

### Shai vs nix/direnv

#### Environment Management

| Feature | Shai | nix/direnv |
|---------|------|------------|
| **Approach** | Container-based | Shell-based |
| **Isolation** | Strong (container) | Weak (shell environment) |
| **Network Control** | Built-in | Not supported |
| **Filesystem** | Container filesystem | Host filesystem |
| **Setup** | Docker required | nix/direnv install |
| **Reproducibility** | High (images) | Very high (nix) |

#### When to Use Each

**Use nix/direnv for:**
- Pure development environments
- Reproducible builds
- No container overhead
- Shell-based workflows

**Use Shai for:**
- AI agent sandboxing
- Network isolation needed
- Filesystem protection required
- Container-based workflows

## Quick Reference

### Use Shai When You Need

- AI agent sandboxing
- Network filtering
- Read-only workspace by default
- Cellular development workflows
- Quick, ephemeral environments
- Fine-grained resource control


## Best Practices

### Use the Right Tool

Don't force Shai where another tool fits better:

- **Development environment**: Devcontainers
- **Production**: Docker/Kubernetes
- **Build pipelines**: Plain Docker
- **AI agents**: Shai ✅

## Learn More

- [Core Concepts](/docs/concepts) - Understanding Shai's design
- [Quick Start](/docs/quick-start) - Get started with Shai
- [Examples](/docs/examples) - Real-world patterns
