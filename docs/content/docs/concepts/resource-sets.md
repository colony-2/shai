---
title: Resource Sets
weight: 2
---

**Resource Sets** are named collections of resources that define what agents can access inside the sandbox.

## What Are Resource Sets?

A resource set is a collection of:
- **HTTP/HTTPS destinations** - Which websites the agent can reach
- **Host mounts** - Which host directories to mount into the container
- **Environment variables** - Which env vars to pass from host to container
- **Ports** - Which TCP/UDP ports to allow (e.g., SSH)
- **Remote calls** - Which host commands can be invoked from inside the sandbox
- **Root commands** - Commands to run as root before switching to the agent user
- **Options** - Container-level options like privileged mode

## Why Resource Sets?

Different coding tasks need different resources:

| Task | Needs |
|------|-------|
| Frontend development | npm registries, CDNs, Playwright |
| Backend development | Database connections, API testing |
| Deployment | Cloud APIs, SSH access, credentials |
| ML/AI development | Model downloads, GPU access |

Resource sets let you define these collections once and reuse them across your project.

## Defining Resource Sets

Resource sets are defined in `.shai/config.yaml`:

```yaml
# .shai/config.yaml
resources:
  base-allowlist:
    http:
      - github.com
      - npmjs.org
      - pypi.org

  frontend-dev:
    http:
      - cdn.jsdelivr.net
      - unpkg.com
    mounts:
      - source: ${{ env.HOME }}/.cache/playwright
        target: /home/${{ conf.TARGET_USER }}/.cache/playwright
        mode: rw

  deployment:
    vars:
      - source: AWS_ACCESS_KEY_ID
    http:
      - amazonaws.com
      - s3.amazonaws.com
    calls:
      - name: deploy-staging
        description: Deploy to staging environment
        command: /usr/local/bin/deploy.sh
        allowed-args: '^--env=staging$'
```

## Resource Set Components

### HTTP/HTTPS Destinations

Allow agents to access specific websites:

```yaml
resources:
  my-tools:
    http:
      - api.openai.com
      - github.com
      - registry.npmjs.org
```

**How it works:**
- Shai configures a proxy (tinyproxy) and DNS filtering (dnsmasq)
- Only listed domains are accessible
- All other network traffic is blocked

{{< callout type="info" >}}
You don't need to specify subdomains separately. `github.com` also allows `api.github.com`, `raw.githubusercontent.com`, etc.
{{< /callout >}}

### Host Mounts

Mount host directories into the container:

```yaml
resources:
  cache-mount:
    mounts:
      - source: ${{ env.HOME }}/.cache/models
        target: /home/${{ conf.TARGET_USER }}/.cache/models
        mode: rw
      - source: /etc/ssl/certs
        target: /etc/ssl/certs
        mode: ro
```

**Fields:**
- `source`: Absolute path on the host
- `target`: Absolute path in the container
- `mode`: `ro` (read-only) or `rw` (read-write)

**Use cases:**
- Sharing package caches (npm, pip, cargo)
- Sharing model weights
- Sharing SSL certificates
- Sharing SSH keys (read-only!)

{{< callout type="warning" >}}
Non-existent source directories are skipped with a warning. This is useful for optional mounts.
{{< /callout >}}

### Environment Variables

Pass environment variables from host to container:

```yaml
resources:
  api-keys:
    vars:
      - source: OPENAI_API_KEY
      - source: ANTHROPIC_API_KEY
```

**Security note:** Variables are *mapped* from host env, not hardcoded in config. This prevents credentials from being committed to git.

{{< callout type="info" >}}
The `target` field is optional. If omitted, the variable keeps the same name in the container. Only specify `target` when you need to rename the variable.
{{< /callout >}}

{{< callout type="error" >}}
**Missing environment variables cause config loading to fail.** This is intentional - you'll catch missing credentials early.
{{< /callout >}}

### Ports

Allow access to specific TCP/UDP ports:

```yaml
resources:
  ssh-access:
    ports:
      - host: github.com
        port: 22
      - host: gitlab.com
        port: 22
```

**Use cases:**
- SSH to git servers
- Connecting to databases
- Accessing custom services

### Remote Calls

Allow agents to invoke specific host commands:

```yaml
resources:
  firmware-dev:
    calls:
      - name: flash-device
        description: Flash firmware to connected device
        command: /usr/local/bin/flash.sh
        allowed-args: '^--port=/dev/tty\w+$'
```

**Fields:**
- `name`: Unique name for the call
- `description`: Human-readable description
- `command`: Absolute path to host command
- `allowed-args`: (Optional) Regex to filter arguments

**Inside the sandbox:**

```bash
# Agent can invoke the call
shai-remote flash-device --port=/dev/ttyUSB0
```

See [Selective Elevation](../selective-elevation) for more details.

### Root Commands

Run commands as root before switching to the agent user:

```yaml
resources:
  docker-in-docker:
    root-commands:
      - "systemctl start docker"
    options:
      privileged: true
```

**Use cases:**
- Starting system services
- Loading kernel modules (`modprobe`)
- Setting up network interfaces

{{< callout type="warning" >}}
Root commands only run when the container starts with root privileges. If the container starts as a non-root user, these commands are skipped.
{{< /callout >}}

### Options

Container-level options:

```yaml
resources:
  gpu-access:
    options:
      privileged: true
```

**Available options:**
- `privileged`: Run container in privileged mode (reduces isolation)

{{< callout type="error" >}}
Use `privileged: true` sparingly! It significantly reduces container security.
{{< /callout >}}

## Template Expansion

Resource set fields support template variables:

```yaml
resources:
  my-tools:
    vars:
      - source: MY_HOST_KEY    # Host env var
        target: API_KEY        # Renamed in container
    mounts:
      - source: ${{ env.HOME }}/.cache
        target: /home/${{ conf.TARGET_USER }}/.cache  # Container user
        mode: rw
```

**Available templates:**
- `${{ env.NAME }}` - Host environment variable
- `${{ vars.NAME }}` - Variable provided via `--var` flag
- `${{ conf.TARGET_USER }}` - The resolved target user (default: `shai`)
- `${{ conf.WORKSPACE }}` - The resolved workspace path (default: `/src`)

## Using Resource Sets

### Via Apply Rules

Resource sets are typically activated via [apply rules](../apply-rules):

```yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: frontend
    resources: [frontend-dev, playwright]

  - path: infrastructure
    resources: [deployment, cloud-apis]
```

When you run `shai -rw frontend/components`, Shai automatically applies `base-allowlist`, `frontend-dev`, and `playwright`.

### Via CLI Flag

You can also opt into resource sets manually:

```bash
shai -rw src --resource-set gpu-access
```

This adds `gpu-access` to whatever resource sets are resolved via apply rules.

## Resource Set Aggregation

When multiple rules match a path, their resource sets are **aggregated** (deduplicated):

```yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: backend
    resources: [database-access]

  - path: backend/payments
    resources: [stripe-api]
```

Running `shai -rw backend/payments` gets all three:
1. `base-allowlist` (from `.`)
2. `database-access` (from `backend`)
3. `stripe-api` (from `backend/payments`)

## Best Practices

### ✅ Do

- Create focused, single-purpose resource sets
- Use descriptive names (`frontend-dev`, not `tools`)
- Map env vars instead of hardcoding credentials
- Document why each resource is needed (via comments)
- Start minimal and add resources as needed

### ❌ Don't

- Create one giant "everything" resource set
- Hardcode secrets in the config
- Grant privileged mode unless absolutely necessary
- Mount sensitive directories as read-write

## Examples

### Frontend Development

```yaml
resources:
  frontend-dev:
    http:
      - cdn.jsdelivr.net
      - fonts.googleapis.com
      - unpkg.com
    mounts:
      - source: ${{ env.HOME }}/.npm
        target: /home/${{ conf.TARGET_USER }}/.npm
        mode: rw
```

### Python ML Development

```yaml
resources:
  ml-dev:
    http:
      - huggingface.co
      - files.pythonhosted.org
    mounts:
      - source: ${{ env.HOME }}/.cache/huggingface
        target: /home/${{ conf.TARGET_USER }}/.cache/huggingface
        mode: rw
    vars:
      - source: HUGGINGFACE_TOKEN
```

### Kubernetes Deployment

```yaml
resources:
  k8s-deploy:
    vars:
      - source: KUBECONFIG
    mounts:
      - source: ${{ env.HOME }}/.kube
        target: /home/${{ conf.TARGET_USER }}/.kube
        mode: ro
    http:
      - kubernetes.default.svc
    calls:
      - name: apply-manifests
        description: Apply Kubernetes manifests
        command: /usr/local/bin/kubectl-apply.sh
        allowed-args: '^--namespace=\w+$'
```

## Next Steps

- Learn about [Apply Rules](../apply-rules) to activate resource sets based on workspace paths
- See [Configuration Reference](/docs/configuration) for complete schema documentation
- Browse [Examples](/docs/examples) for real-world resource set patterns
