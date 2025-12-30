---
title: Custom Images
weight: 3
---

Build custom Docker images tailored to your specific development needs.

## Why Custom Images?

Build a custom image when you need:
- Specific tool versions not in shai-mega
- Specialized development tools
- GPU support for ML/AI
- Embedded development toolchains
- Minimal footprint with only your tools
- Company-specific tooling

## Requirements

All Shai-compatible images must include these packages:

### Required System Packages

```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends \
    supervisor \
    dnsmasq \
    iptables \
    tinyproxy \
    bash \
    ca-certificates \
    coreutils \
    curl \
    iproute2 \
    iputils-ping \
    jq \
    net-tools \
    passwd \
    procps \
    sed \
    util-linux \
    && rm -rf /var/lib/apt/lists/*
```

{{< callout type="info" >}}
**Shortcut:** Base your image on `ghcr.io/colony-2/shai-base:latest` which includes all requirements.
{{< /callout >}}

### Supervisord Installation

Shai requires supervisord to be installed. This is automatic if you base on shai-base.

If building from scratch, ensure supervisord is installed and in PATH. The bootstrap process automatically starts supervisord and loads service configurations from `/etc/supervisor/conf.d/*.conf`.

## Building from shai-base

### Example: Python ML Development

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Python and ML tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.11 \
    python3-pip \
    python3-dev \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install ML frameworks
RUN pip3 install --no-cache-dir \
    numpy \
    pandas \
    scikit-learn \
    torch \
    transformers \
    jupyter

# Install development tools
RUN pip3 install --no-cache-dir \
    black \
    ruff \
    mypy \
    pytest

WORKDIR /src
```

### Example: Go Development

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Go 1.21
ARG GO_VERSION=1.21.6
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz | \
    tar -C /usr/local -xzf -

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"

# Install Go tools
RUN go install golang.org/x/tools/gopls@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /src
```

### Example: Rust Development

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
    sh -s -- -y --default-toolchain stable --profile minimal

# Install Rust tools
RUN cargo install cargo-watch cargo-edit cargo-audit

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
```

### Example: Node.js Specific Version

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install Node.js 18 LTS
RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install pnpm
RUN npm install -g pnpm

# Install global tools
RUN pnpm install -g \
    typescript \
    tsx \
    @nestjs/cli \
    prisma

WORKDIR /src
```

## Specialized Images

### GPU Support (PyTorch)

```dockerfile
FROM nvidia/cuda:12.1.0-cudnn8-devel-ubuntu22.04

# Install Shai requirements first
RUN apt-get update && apt-get install -y --no-install-recommends \
    supervisor \
    dnsmasq \
    iptables \
    tinyproxy \
    bash \
    ca-certificates \
    coreutils \
    curl \
    iproute2 \
    iputils-ping \
    jq \
    net-tools \
    passwd \
    procps \
    sed \
    util-linux \
    python3.10 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install PyTorch with CUDA support
RUN pip3 install --no-cache-dir \
    torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121

# Install ML tools
RUN pip3 install --no-cache-dir \
    transformers \
    accelerate \
    datasets \
    wandb

WORKDIR /src
```

**Usage:**
```yaml
# .shai/config.yaml
apply:
  - path: ml/training
    image: ghcr.io/my-org/pytorch-gpu:latest
    resources: [gpu-access]
```

### Embedded Development (ARM)

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install ARM toolchain
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc-arm-none-eabi \
    gdb-multiarch \
    openocd \
    picocom \
    && rm -rf /var/lib/apt/lists/*

# Install Rust for embedded
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH

RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
    sh -s -- -y --default-toolchain stable

# Add ARM targets
RUN rustup target add thumbv7em-none-eabihf thumbv6m-none-eabi

# Install cargo-embed and probe-rs
RUN cargo install cargo-embed probe-rs

WORKDIR /src
```

### DevOps Tools

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

# Install cloud CLIs
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf aws awscliv2.zip

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Terraform
ARG TERRAFORM_VERSION=1.6.0
RUN curl -fsSL https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip -o terraform.zip && \
    unzip terraform.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform.zip

# Install Pulumi
RUN curl -fsSL https://get.pulumi.com | sh

ENV PATH="/root/.pulumi/bin:${PATH}"

WORKDIR /src
```

## Multi-Stage Builds

Reduce final image size with multi-stage builds:

```dockerfile
# Stage 1: Build dependencies
FROM ghcr.io/colony-2/shai-base:latest as builder

RUN apt-get update && apt-get install -y build-essential
# ... build steps ...

# Stage 2: Runtime
FROM ghcr.io/colony-2/shai-base:latest

# Copy only what's needed
COPY --from=builder /path/to/binaries /usr/local/bin/

WORKDIR /src
```

## Best Practices

### ✅ Do

1. **Base on shai-base** for automatic requirements:
   ```dockerfile
   FROM ghcr.io/colony-2/shai-base:latest
   ```

2. **Clean up apt lists** to reduce size:
   ```dockerfile
   RUN apt-get update && apt-get install -y package \
       && rm -rf /var/lib/apt/lists/*
   ```

3. **Use --no-install-recommends** to avoid bloat:
   ```dockerfile
   RUN apt-get install -y --no-install-recommends package
   ```

4. **Combine RUN commands** to reduce layers:
   ```dockerfile
   RUN apt-get update && \
       apt-get install -y pkg1 pkg2 && \
       rm -rf /var/lib/apt/lists/*
   ```

5. **Version pin critical tools**:
   ```dockerfile
   ARG GO_VERSION=1.21.6
   RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
   ```

6. **Document requirements** in README
7. **Test with Shai** before publishing

### ❌ Don't

1. **Don't forget Shai requirements** if not basing on shai-base
2. **Don't leave apt lists** (`/var/lib/apt/lists/*`)
3. **Don't run as root** (Shai handles user creation)
4. **Don't hardcode secrets** in the image
5. **Don't use latest tags** for critical dependencies
6. **Don't include sensitive data** in layers

## Building and Publishing

### Build Locally

```bash
docker build -t my-org/my-shai-image:latest .
```

### Test with Shai

```bash
shai --image my-org/my-shai-image:latest -rw src --verbose
```

### Publish to Registry

```bash
# GitHub Container Registry
docker tag my-org/my-shai-image:latest ghcr.io/my-org/my-shai-image:latest
docker push ghcr.io/my-shai-image:latest

# Docker Hub
docker tag my-org/my-shai-image:latest my-org/my-shai-image:latest
docker push my-org/my-shai-image:latest
```

### Use in Configuration

```yaml
# .shai/config.yaml
type: shai-sandbox
version: 1
image: ghcr.io/my-org/my-shai-image:latest
```

## Testing Custom Images

### Verify Requirements

```bash
# Check required packages
shai --image my-image:latest -- bash -c "
  which supervisord && \
  which dnsmasq && \
  which iptables && \
  which tinyproxy && \
  echo 'All requirements present'
"
```

### Test Sandboxing

```bash
# Verify network filtering works
shai --image my-image:latest --verbose -rw . -- bash -c "
  cat /var/log/shai/iptables.out
"
```

### Test Your Tools

```bash
# Verify your custom tools
shai --image my-image:latest -- bash -c "
  go version && \
  python3 --version && \
  cargo --version
"
```

## Example: Complete Custom Image

```dockerfile
FROM ghcr.io/colony-2/shai-base:latest

LABEL org.opencontainers.image.source=https://github.com/my-org/my-repo
LABEL org.opencontainers.image.description="Custom Shai image for XYZ project"

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    pkg-config \
    libssl-dev \
    git \
    vim \
    && rm -rf /var/lib/apt/lists/*

# Install Python 3.11
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.11 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages
RUN pip3 install --no-cache-dir \
    fastapi \
    uvicorn \
    sqlalchemy \
    alembic \
    pytest \
    black \
    mypy

# Install Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install global npm tools
RUN npm install -g typescript prettier eslint

# Set working directory
WORKDIR /src

# Health check (optional)
HEALTHCHECK --interval=30s --timeout=3s \
  CMD which supervisord || exit 1
```

## Troubleshooting

### Shai fails to start

**Check:** Are all required packages present?

```bash
docker run --rm my-image:latest bash -c "which supervisord dnsmasq iptables tinyproxy"
```

### Network filtering doesn't work

**Check:** iptables and networking tools installed

```bash
docker run --rm my-image:latest bash -c "iptables --version"
```

### Permission issues

**Check:** Don't run commands as a specific user in the Dockerfile - Shai handles user creation

## Next Steps

- Review [shai-base](../shai-base) as a starting point
- See [Configuration](/docs/configuration) for using custom images
- Browse [Examples](/docs/examples) for real-world patterns
