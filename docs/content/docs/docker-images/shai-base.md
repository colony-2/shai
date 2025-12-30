---
title: shai-base
weight: 1
---

Minimal Debian-based image containing only the essential packages required for Shai sandboxing.

## Overview

`shai-base` provides the bare minimum infrastructure needed for Shai to function:
- Sandboxing utilities (supervisord, dnsmasq, iptables, tinyproxy)
- Core system utilities
- No language runtimes
- No development tools

**Registry:** `ghcr.io/colony-2/shai-base:latest`

**Base:** `debian:bookworm-slim`

**Size:** ~200 MB

## What's Included

### Sandboxing Tools
- **supervisor** - Process supervisor for managing background services
- **dnsmasq** - DNS server for domain filtering
- **iptables** - Firewall for network egress control
- **tinyproxy** - HTTP/HTTPS proxy for allow-listed traffic

### System Utilities
- **bash** - Shell
- **ca-certificates** - SSL/TLS certificates
- **coreutils** - Core Unix utilities (ls, cp, mv, etc.)
- **curl** - HTTP client
- **iproute2** - Network configuration (ip command)
- **iputils-ping** - Network testing (ping)
- **jq** - JSON processor
- **net-tools** - Network utilities (netstat, etc.)
- **passwd** - User management
- **procps** - Process utilities (ps, top, etc.)
- **sed** - Stream editor
- **util-linux** - System utilities (mount, etc.)

## Use Cases

### 1. Building Custom Images

`shai-base` is ideal as a foundation for custom development images:

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Python
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install Python tools
RUN pip3 install --no-cache-dir \
    black \
    mypy \
    pytest
```

### 2. Fast CI/CD

Smaller images mean faster pulls and startup:

```yaml
# .github/workflows/test.yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          shai --image ghcr.io/colony-2/shai-base:latest -- ./run-tests.sh
```

### 3. Minimal Overhead

When you need the lightest possible container:

```yaml
# .shai/config.yaml
image: ghcr.io/colony-2/shai-base:latest
```

### 4. Security-Sensitive Environments

Fewer packages mean smaller attack surface:
- No unnecessary tools installed
- Minimal dependencies
- Easier to audit

## Configuration Example

```yaml
# .shai/config.yaml
type: shai-sandbox
version: 1

# Use shai-base
image: ghcr.io/colony-2/shai-base:latest

resources:
  base-allowlist:
    http:
      - github.com
      - npmjs.org

apply:
  - path: ./
    resources: [base-allowlist]
```

## Extending shai-base

### Example: Python Development

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Python and common tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.11 \
    python3-pip \
    python3-venv \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Python development tools
RUN pip3 install --no-cache-dir \
    black \
    ruff \
    mypy \
    pytest \
    ipython

WORKDIR /src
```

### Example: Node.js Development

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install global npm tools
RUN npm install -g \
    typescript \
    eslint \
    prettier

WORKDIR /src
```

### Example: Go Development

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Go 1.21
RUN curl -fsSL https://go.dev/dl/go1.21.6.linux-amd64.tar.gz | \
    tar -C /usr/local -xzf -

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/shai/go"

# Install Go tools
RUN go install golang.org/x/tools/gopls@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

WORKDIR /src
```

## Building Custom Images

### Basic Build

```bash
# Create Dockerfile
cat > Dockerfile.custom <<'EOF'
FROM ghcr.io/colony-2/shai-base:latest
RUN apt-get update && apt-get install -y python3
EOF

# Build
docker build -f Dockerfile.custom -t my-shai-image:latest .

# Use with Shai
shai --image my-shai-image:latest -rw src
```

### Multi-Stage Build

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest as builder

# Build dependencies
RUN apt-get update && apt-get install -y build-essential
# ... build steps ...

FROM ghcr.io/colony-2/shai-base:latest

# Copy built artifacts
COPY --from=builder /build/output /usr/local/bin/

WORKDIR /src
```

## Limitations

### What's NOT Included

- Language runtimes (Go, Rust, Node, Python, Java)
- Development tools (git, vim, etc.)
- AI CLI tools (claude-code, codex, etc.)
- Build tools (make, gcc, etc.)
- Package managers beyond system apt

### When shai-base Isn't Enough

If you need:
- **Multiple languages**: Use [shai-mega](../shai-mega) instead
- **AI tools pre-installed**: Use [shai-mega](../shai-mega)
- **Quick start**: Use [shai-mega](../shai-mega)
- **Specialized tools**: Build a [custom image](../custom)

## Performance

### Startup Time
- **Cold start** (first pull): ~30 seconds
- **Warm start** (cached): ~1 second

### Resource Usage
- **Disk**: ~200 MB
- **Memory**: ~50 MB (sandboxing overhead only)
- **CPU**: Minimal overhead

## Maintenance

### Updating

Pull the latest version:

```bash
docker pull ghcr.io/colony-2/shai-base:latest
```

### Versioning

Tags available:
- `latest` - Latest stable release (recommended)
- `v1.0.0` - Specific version (when pinning is needed)

## Troubleshooting

### Missing Tools

**Problem:** Tool not found in shai-base

**Solution:** Extend the image or use shai-mega

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest
RUN apt-get update && apt-get install -y <your-tool>
```

### Slow Builds

**Problem:** Building custom image is slow

**Solution:** Use BuildKit and layer caching

```bash
DOCKER_BUILDKIT=1 docker build --cache-from ghcr.io/colony-2/shai-base:latest ...
```

## Best Practices

### ✅ Do

- Use as a base for custom images
- Keep custom images minimal
- Cache apt packages properly
- Document required tools in Dockerfile
- Version your custom images

### ❌ Don't

- Install everything into shai-base manually
- Skip cleanup steps (`rm -rf /var/lib/apt/lists/*`)
- Forget to update package lists before install
- Use `latest` tag in production (pin versions instead)

## Next Steps

- Learn how to build [Custom Images](../custom)
- Compare with [shai-mega](../shai-mega)
- See [Configuration Reference](/docs/configuration) for image settings
