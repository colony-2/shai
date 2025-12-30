---
title: Docker Images
weight: 4
---

Shai works with Docker images that include the required sandboxing utilities.

## Overview

Shai provides two official images:

1. **[shai-base](shai-base)** - Minimal image with only sandboxing essentials
2. **[shai-mega](shai-mega)** - Full-featured development environment (default)

You can also create [custom images](custom) based on your needs.

## Quick Comparison

| Feature | shai-base | shai-mega |
|---------|-----------|-----------|
| **Size** | ~200 MB | ~2 GB |
| **Startup** | Fast | Moderate |
| **Languages** | None | Go, Rust, Node, Python, Java |
| **AI Tools** | None | Claude Code, Codex, Gemini CLI |
| **Use Case** | Custom images, CI/CD | Quick start, full development |

## Which Image Should I Use?

### Use `shai-mega` if:
- You're just getting started with Shai
- You work with multiple languages
- You want AI tools pre-installed
- Disk space isn't a concern

### Use `shai-base` if:
- You're building custom images
- You need fast startup (CI/CD)
- You want minimal overhead
- You have specific tooling requirements

### Use a custom image if:
- You need specialized tools
- You have specific version requirements
- You're working with embedded systems
- You need GPU support

## Image Registry

Both official images are hosted on GitHub Container Registry:

```bash
# Pull shai-base
docker pull ghcr.io/colony-2/shai-base:latest

# Pull shai-mega
docker pull ghcr.io/colony-2/shai-mega:latest
```

## Configuration

Specify the image in `.shai/config.yaml`:

```yaml
type: shai-sandbox
version: 1

# Use shai-mega (default)
image: ghcr.io/colony-2/shai-mega

# Or use shai-base
image: ghcr.io/colony-2/shai-base

# Or use a custom image
image: ghcr.io/my-org/custom-dev:latest
```

## CLI Override

Override the image at runtime:

```bash
shai -rw src --image ghcr.io/my-org/custom:latest
```

## Requirements

All Shai-compatible images must include:

### Required Packages
- **supervisord** - Process supervisor
- **dnsmasq** - DNS server for domain filtering
- **iptables** - Firewall for network egress control
- **tinyproxy** - HTTP/HTTPS proxy
- **bash, coreutils, iproute2, iputils-ping, jq, net-tools, passwd, procps, sed, util-linux**

Both official images include these by default.

## Learn More

{{< cards >}}
  {{< card link="shai-base" title="shai-base" icon="cube" subtitle="Minimal sandboxing foundation" >}}
  {{< card link="shai-mega" title="shai-mega" icon="cube" subtitle="Full development environment" >}}
  {{< card link="custom" title="Custom Images" icon="document-text" subtitle="Build your own images" >}}
{{< /cards >}}
