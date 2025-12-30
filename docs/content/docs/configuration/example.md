---
title: Complete Example
weight: 3
---

A fully annotated `.shai/config.yaml` demonstrating all features.

## Full Configuration

```yaml
# ==============================================================================
# Shai Configuration File
# ==============================================================================

# Required: Schema type identifier
type: shai-sandbox

# Required: Schema version
version: 1

# Required: Base container image
# Use shai-mega for kitchen-sink dev environment
# or shai-base for minimal footprint
image: ghcr.io/colony-2/shai-mega

# Optional: Default user inside container (default: shai)
# This user is created if it doesn't exist, with UID/GID matched to host user
user: shai

# Optional: Workspace path inside container (default: /src)
# Most configs should stick with /src
workspace: /src

# ==============================================================================
# Resource Sets
# ==============================================================================
# Define named collections of resources that can be applied via apply rules

resources:
  # --------------------------------------------------------------------------
  # Base Allowlist
  # --------------------------------------------------------------------------
  # Common package registries and APIs - applied to all paths
  base-allowlist:
    http:
      - github.com
      - npmjs.org
      - registry.npmjs.org
      - pypi.org
      - files.pythonhosted.org
      - crates.io
      - golang.org
      - proxy.golang.org

  # --------------------------------------------------------------------------
  # Git SSH Access
  # --------------------------------------------------------------------------
  # SSH access to git servers for cloning/pushing
  git-ssh:
    ports:
      - host: github.com
        port: 22
      - host: gitlab.com
        port: 22

  # --------------------------------------------------------------------------
  # Frontend Development
  # --------------------------------------------------------------------------
  # Resources for React/Vue/Svelte development
  frontend-dev:
    http:
      - cdn.jsdelivr.net
      - unpkg.com
      - fonts.googleapis.com
      - fonts.gstatic.com

    mounts:
      # npm cache (read-write)
      - source: ${{ env.HOME }}/.npm
        target: /home/${{ conf.TARGET_USER }}/.npm
        mode: rw

      # playwright browsers (read-write for installs)
      - source: ${{ env.HOME }}/.cache/playwright
        target: /home/${{ conf.TARGET_USER }}/.cache/playwright
        mode: rw

  # --------------------------------------------------------------------------
  # Backend Development
  # --------------------------------------------------------------------------
  # Resources for Go/Rust/Node backend services
  backend-dev:
    http:
      - pkg.go.dev
      - sum.golang.org

    mounts:
      # Go module cache
      - source: ${{ env.HOME }}/go/pkg/mod
        target: /home/${{ conf.TARGET_USER }}/go/pkg/mod
        mode: rw

      # Cargo registry cache
      - source: ${{ env.HOME }}/.cargo/registry
        target: /home/${{ conf.TARGET_USER }}/.cargo/registry
        mode: rw

  # --------------------------------------------------------------------------
  # Database Access
  # --------------------------------------------------------------------------
  # For services that need database connectivity
  database-access:
    vars:
      # Map database URL from host environment
      - source: DATABASE_URL

    ports:
      # PostgreSQL
      - host: localhost
        port: 5432

      # Redis
      - host: localhost
        port: 6379

  # --------------------------------------------------------------------------
  # ML/AI Development
  # --------------------------------------------------------------------------
  # Resources for machine learning workflows
  ml-dev:
    http:
      - huggingface.co
      - cdn.huggingface.co
      - pypi.org
      - files.pythonhosted.org

    vars:
      # Hugging Face token for model downloads
      - source: HUGGINGFACE_TOKEN

    mounts:
      # Model cache (can be very large)
      - source: ${{ env.HOME }}/.cache/huggingface
        target: /home/${{ conf.TARGET_USER }}/.cache/huggingface
        mode: rw

      # pip cache
      - source: ${{ env.HOME }}/.cache/pip
        target: /home/${{ conf.TARGET_USER }}/.cache/pip
        mode: rw

  # --------------------------------------------------------------------------
  # Cloud Deployment
  # --------------------------------------------------------------------------
  # For Pulumi, Terraform, and cloud CLI tools
  cloud-deployment:
    http:
      - api.pulumi.com
      - amazonaws.com
      - s3.amazonaws.com
      - cloudfront.net

    vars:
      # AWS credentials
      - source: AWS_ACCESS_KEY_ID
      - source: AWS_SECRET_ACCESS_KEY
      - source: AWS_REGION

      # Pulumi access token
      - source: PULUMI_ACCESS_TOKEN

    mounts:
      # AWS config (read-only)
      - source: ${{ env.HOME }}/.aws
        target: /home/${{ conf.TARGET_USER }}/.aws
        mode: ro

    calls:
      # Remote call to verify deployment
      - name: verify-deployment
        description: Verify deployment succeeded
        command: /usr/local/bin/verify-deployment.sh
        allowed-args: '^--stack=[\w-]+ --env=(staging|production)$'

  # --------------------------------------------------------------------------
  # Kubernetes Access
  # --------------------------------------------------------------------------
  # For kubectl, helm, and k8s deployments
  k8s-access:
    vars:
      - source: KUBECONFIG

    mounts:
      # Kubernetes config (read-only to prevent accidental modifications)
      - source: ${{ env.HOME }}/.kube
        target: /home/${{ conf.TARGET_USER }}/.kube
        mode: ro

    http:
      - kubernetes.default.svc

  # --------------------------------------------------------------------------
  # Docker-in-Docker
  # --------------------------------------------------------------------------
  # For agents that need to build/run Docker containers
  docker-in-docker:
    mounts:
      # Docker socket (allows controlling host Docker)
      - source: /var/run/docker.sock
        target: /var/run/docker.sock
        mode: rw

    root-commands:
      # Ensure Docker service is running
      - "systemctl start docker || true"

    options:
      # Required for Docker-in-Docker
      privileged: true

  # --------------------------------------------------------------------------
  # Embedded Development
  # --------------------------------------------------------------------------
  # For embedded systems and firmware development
  embedded-dev:
    http:
      - developer.arm.com

    mounts:
      # USB device access
      - source: /dev
        target: /dev
        mode: rw

    calls:
      # Flash firmware to connected device
      - name: flash-device
        description: Flash compiled firmware to USB device
        command: /usr/local/bin/flash-firmware.sh
        allowed-args: '^--device=/dev/ttyUSB[0-9]+ --firmware=/tmp/[\w-]+\.hex$'

    options:
      # Required for device access
      privileged: true

# ==============================================================================
# Apply Rules
# ==============================================================================
# Map workspace paths to resource sets
# Rules are evaluated top-to-bottom; all matching rules are aggregated

apply:
  # --------------------------------------------------------------------------
  # Root - Applies to Everything
  # --------------------------------------------------------------------------
  - path: ./
    resources:
      - base-allowlist
      - git-ssh

  # --------------------------------------------------------------------------
  # Frontend Applications
  # --------------------------------------------------------------------------
  - path: packages/web-app
    resources:
      - frontend-dev

  - path: packages/mobile-app
    resources:
      - frontend-dev

  - path: packages/marketing-site
    resources:
      - frontend-dev

  # --------------------------------------------------------------------------
  # Backend Services
  # --------------------------------------------------------------------------
  - path: services/api
    resources:
      - backend-dev
      - database-access

  - path: services/auth
    resources:
      - backend-dev
      - database-access

  - path: services/workers
    resources:
      - backend-dev
      - database-access

  # --------------------------------------------------------------------------
  # Machine Learning
  # --------------------------------------------------------------------------
  - path: ml/training
    resources:
      - ml-dev
    # Use GPU-enabled image for training
    image: ghcr.io/my-org/pytorch-gpu:latest

  - path: ml/inference
    resources:
      - ml-dev

  # --------------------------------------------------------------------------
  # Infrastructure
  # --------------------------------------------------------------------------
  - path: infrastructure/staging
    resources:
      - cloud-deployment
      - k8s-access
    # Use DevOps image with cloud CLI tools
    image: ghcr.io/my-org/devops:latest

  - path: infrastructure/production
    resources:
      - cloud-deployment
      - k8s-access
    image: ghcr.io/my-org/devops:latest

  # --------------------------------------------------------------------------
  # Container Builds
  # --------------------------------------------------------------------------
  - path: docker
    resources:
      - docker-in-docker

  # --------------------------------------------------------------------------
  # Embedded Firmware
  # --------------------------------------------------------------------------
  - path: firmware
    resources:
      - embedded-dev
    # Use embedded toolchain image
    image: ghcr.io/my-org/embedded-toolchain:latest
```

## Usage Examples

### Frontend Development

```bash
# Work on web app
shai -rw packages/web-app

# Automatic resources: base-allowlist, git-ssh, frontend-dev
# Can access: npm, CDNs, fonts, etc.
# npm cache is shared with host
```

### Backend API Development

```bash
# Work on API service
shai -rw services/api

# Automatic resources: base-allowlist, git-ssh, backend-dev, database-access
# Can access: Go modules, databases, etc.
# Database credentials from environment
```

### ML Training

```bash
# Train models (uses GPU image)
shai -rw ml/training

# Automatic resources: base-allowlist, git-ssh, ml-dev
# Uses: pytorch-gpu:latest image
# Hugging Face cache shared with host
```

### Infrastructure Deployment

```bash
# Deploy to staging
shai -rw infrastructure/staging

# Automatic resources: base-allowlist, git-ssh, cloud-deployment, k8s-access
# Uses: devops:latest image
# AWS credentials from environment
# Can call: verify-deployment remote command
```

### Docker Image Building

```bash
# Build Docker images
shai -rw docker

# Automatic resources: base-allowlist, git-ssh, docker-in-docker
# Has access to host Docker daemon
# Runs in privileged mode
```

### Embedded Firmware Development

```bash
# Flash firmware
shai -rw firmware

# Automatic resources: base-allowlist, git-ssh, embedded-dev
# Uses: embedded-toolchain:latest image
# Can call: flash-device remote command
# Has access to USB devices
```

## Customization

### Override Resource Sets

Add additional resource sets at runtime:

```bash
shai -rw services/api --resource-set ml-dev
# Gets: backend-dev, database-access, AND ml-dev
```

### Override Image

Use a different image:

```bash
shai -rw ml/training --image ghcr.io/my-org/custom:latest
```

### Provide Variables

Pass variables for templates:

```bash
shai --var ENV=staging --var REGION=us-east-1 -rw infrastructure
```

## Notes

### Security Considerations

1. **Privileged mode** used sparingly (only docker-in-docker and embedded-dev)
2. **Credentials** never hardcoded - always from environment
3. **Read-only mounts** for sensitive data (.kube, .aws)
4. **Strict argument validation** for remote calls
5. **Minimal resource grants** per path

### Performance Optimizations

1. **Cache mounts** shared with host (npm, cargo, pip, etc.)
2. **Model caches** prevent re-downloading large files
3. **Specific images** per use case (GPU for ML, DevOps for infrastructure)

### Maintenance Tips

1. Document required environment variables in README
2. Keep resource sets focused and single-purpose
3. Use descriptive names for calls and resource sets
4. Review apply rules periodically as project evolves
5. Test configs with `--verbose` to debug

## Next Steps

- Review [Schema Reference](schema) for field details
- Learn about [Template Expansion](templates) for dynamic configs
- Browse [Examples](/docs/examples) for more patterns
