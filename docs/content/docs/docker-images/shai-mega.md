---
title: shai-mega
weight: 2
---

Comprehensive development environment with multiple language runtimes, tools, and pre-installed AI CLIs.

## Overview

`shai-mega` is a kitchen-sink image designed to get you started quickly with Shai. It includes everything you need for polyglot development with AI agents.

**Registry:** `ghcr.io/colony-2/shai-mega:latest`

**Base:** `ghcr.io/colony-2/shai-base:latest` (Debian bookworm-slim)

**Size:** ~2 GB

**Default:** This is Shai's default image when no config is specified

## What's Included

### Languages & Runtimes

- **Go** - Latest stable (1.21+)
- **Rust** - Stable toolchain with cargo
- **Node.js** - Latest LTS with npm, yarn, pnpm
- **Python** - Python 3.11+ with pip
- **Java** - OpenJDK (default-jdk)
- **C/C++** - GCC and Clang compilers

### Package Managers

- **npm** - Node.js package manager
- **yarn** - Alternative Node.js package manager
- **pnpm** - Fast, disk-efficient Node.js package manager
- **pip** - Python package installer
- **cargo** - Rust package manager

### AI CLI Tools

- **Claude Code** - Anthropic's official CLI
- **Codex** - OpenAI's coding agent
- **Gemini CLI** - Google's AI CLI
- **Moonrepo** - Repository management tool

### Browser Automation

- **Playwright** - Browser testing framework
- **Chromium** - Headless browser

### Development Tools

- **git** - Version control
- **jq** - JSON processor
- **curl** - HTTP client
- **wget** - File downloader
- **bash-completion** - Shell completions
- **vim** - Text editor
- **nano** - Simple text editor
- **htop** - Process viewer
- **tree** - Directory tree viewer
- **rsync** - File synchronization
- **ssh** - Secure shell client

### Build Tools

- **build-essential** - Essential build tools (make, gcc, etc.)
- **pkg-config** - Library configuration helper
- **cmake** - Cross-platform build system

### System Tools

All tools from [shai-base](../shai-base) plus:
- **zsh** - Advanced shell
- **tmux** - Terminal multiplexer
- **supervisor** - Process manager
- **iptables** - Firewall
- **tinyproxy** - HTTP proxy
- **dnsmasq** - DNS server

## Use Cases

### 1. Quick Start

Get started with Shai immediately without building custom images:

```bash
npm install -g @colony2/shai
shai -rw src -- claude
```

### 2. Polyglot Development

Work on projects using multiple languages:

```bash
# Monorepo with Go backend + Node.js frontend
shai -rw services/api -rw packages/web-app
```

### 3. AI Agent Development

Pre-installed AI tools mean agents can run immediately:

```bash
# Claude Code is already installed
shai -rw . -- claude

# Or Gemini CLI
shai -rw . -- gemini-cli
```

### 4. Full-Stack Development

Everything you need for modern web development:

```bash
# React frontend + Node.js backend + Python ML
shai -rw frontend -rw backend -rw ml
```

## Configuration Example

```yaml
# .shai/config.yaml
type: shai-sandbox
version: 1

# shai-mega is the default, but can be explicit
image: ghcr.io/colony-2/shai-mega

resources:
  base-allowlist:
    http:
      - github.com
      - npmjs.org
      - pypi.org
      - crates.io

apply:
  - path: ./
    resources: [base-allowlist]
```

## Pre-Installed Tools Locations

All language toolchains are installed system-wide:

```bash
# Go
/usr/local/go/bin/go

# Rust
/usr/local/cargo/bin/cargo
/usr/local/cargo/bin/rustc

# Node.js
/usr/bin/node
/usr/bin/npm

# Python
/usr/bin/python3
/usr/bin/pip3

# AI Tools
/usr/local/bin/claude
/usr/local/bin/gemini-cli
```

PATH is pre-configured for all users.

## Version Information

Check installed versions:

```bash
shai -- bash -c "
  echo Go: \$(go version)
  echo Rust: \$(rustc --version)
  echo Node: \$(node --version)
  echo Python: \$(python3 --version)
"
```

## Performance Considerations

### Startup Time

- **Cold start** (first pull): 2-5 minutes (large download)
- **Warm start** (cached): ~2 seconds

{{< callout type="info" >}}
**Pre-warm the image** for faster first run:
```bash
docker pull ghcr.io/colony-2/shai-mega:latest
```
{{< /callout >}}

### Resource Usage

- **Disk**: ~2 GB
- **Memory**: 100-200 MB (sandboxing) + your application
- **CPU**: Minimal overhead

### Optimization Tips

1. **Use cache mounts** to avoid re-downloading packages:
   ```yaml
   resources:
     caches:
       mounts:
         - source: ${{ env.HOME }}/.npm
           target: /home/shai/.npm
           mode: rw
         - source: ${{ env.HOME }}/.cargo
           target: /home/shai/.cargo
           mode: rw
   ```

2. **Pre-install dependencies** in your project's setup scripts

3. **Use shai-base** for CI/CD where you don't need all tools

## When to Use shai-mega

### ✅ Use shai-mega if:

- You're just getting started with Shai
- You work on multiple projects with different languages
- You want AI tools pre-installed
- You don't want to manage custom images
- Disk space isn't a concern
- You value convenience over image size

### ❌ Don't use shai-mega if:

- You need a minimal image (use [shai-base](../shai-base))
- You're running in resource-constrained CI/CD
- You have specific version requirements (build [custom](../custom))
- You need specialized tools not included
- You need GPU support (build custom from shai-base)
- Image size is critical

## Updating

### Pull Latest Version

```bash
docker pull ghcr.io/colony-2/shai-mega:latest
```

### Check for Updates

```bash
# See current image ID
docker images ghcr.io/colony-2/shai-mega

# Pull and compare
docker pull ghcr.io/colony-2/shai-mega:latest
```

## Versioning

Available tags:
- `latest` - Latest stable release (recommended)
- `v1.0.0` - Specific version (for reproducibility)
- `main` - Latest build from main branch (unstable)

**Recommendation:** Use `latest` for development, pin versions in CI/CD:

```yaml
# Development
image: ghcr.io/colony-2/shai-mega

# CI/CD
image: ghcr.io/colony-2/shai-mega:v1.0.0
```

## Customizing shai-mega

You can extend shai-mega if you need additional tools:

```dockerfile
FROM ghcr.io/colony-2/shai-mega:latest

# Add additional tools
RUN apt-get update && apt-get install -y \
    postgresql-client \
    redis-tools \
    && rm -rf /var/lib/apt/lists/*

# Install additional npm packages globally
RUN npm install -g \
    typescript \
    @nestjs/cli

WORKDIR /src
```

## Troubleshooting

### Image Pull Timeout

**Problem:** Pulling shai-mega times out

**Solution:** The image is large (~2 GB). Ensure stable network:

```bash
# Increase Docker pull timeout
DOCKER_CLI_TIMEOUT=600 docker pull ghcr.io/colony-2/shai-mega:latest
```

### Tool Version Conflicts

**Problem:** Need a specific version not in shai-mega

**Solution:** Build a [custom image](../custom) based on shai-base

### Missing Tool

**Problem:** Tool you need isn't included

**Solution:** Either extend shai-mega or build from shai-base:

```dockerfile
FROM ghcr.io/colony-2/shai-mega:latest
RUN apt-get update && apt-get install -y your-tool
```

## Comparison with shai-base

| Feature | shai-base | shai-mega |
|---------|-----------|-----------|
| **Size** | ~200 MB | ~2 GB |
| **Languages** | None | 6+ languages |
| **AI Tools** | None | 3+ tools |
| **Startup** | Fast | Moderate |
| **Use Case** | Custom images, CI | Quick start, development |
| **Flexibility** | Maximum | High (but opinionated) |

## Next Steps

- Compare with [shai-base](../shai-base) for minimal images
- Learn to build [Custom Images](../custom) for specialized needs
- See [Examples](/docs/examples) for real-world usage
