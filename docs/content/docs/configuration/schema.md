---
title: Schema Reference
weight: 1
---

Complete schema documentation for `.shai/config.yaml`.

{{< callout type="info" >}}
**See also:** [Default configuration file](https://github.com/colony-2/shai/blob/main/internal/shai/runtime/config/shai.default.yaml) used when no custom config exists.
{{< /callout >}}

## Top-Level Keys

### `type`

**Required:** Yes
**Type:** String
**Value:** `shai-sandbox`

Schema identifier for validation.

```yaml
type: shai-sandbox
```

---

### `version`

**Required:** Yes
**Type:** Integer
**Value:** `1`

Configuration schema version. Future releases may bump this.

```yaml
version: 1
```

---

### `image`

**Required:** Yes
**Type:** String

Base container image to use for sandboxes.

```yaml
image: ghcr.io/colony-2/shai-mega
```

**Supports templates:** Yes (`${{ env.* }}`, `${{ vars.* }}`)

**Examples:**
```yaml
# Official images
image: ghcr.io/colony-2/shai-base
image: ghcr.io/colony-2/shai-mega

# Custom image
image: ghcr.io/my-org/dev-env:latest

# Using environment variable
image: ${{ env.DEV_IMAGE }}
```

---

### `user`

**Required:** No
**Default:** `shai`
**Type:** String

Container user Shai switches to before running your command.

```yaml
user: developer
```

**Supports templates:** Yes (`${{ env.* }}`, `${{ vars.* }}`)

**CLI override:**
```bash
shai --user myuser
```

{{< callout type="info" >}}
This user is created inside the container if it doesn't exist. The user's UID/GID are matched to your host user for seamless file permissions.
{{< /callout >}}

---

### `workspace`

**Required:** No
**Default:** `/src`
**Type:** String

Absolute path of the repository inside the container.

```yaml
workspace: /workspace
```

**Supports templates:** Yes (`${{ env.* }}`, `${{ vars.* }}`)

{{< callout type="warning" >}}
Most configurations should stick with the default `/src`. Only change this if you have a specific reason.
{{< /callout >}}

---

### `resources`

**Required:** Yes
**Type:** Map of resource set definitions

Defines named resource sets. See [Resource Sets](#resource-sets) below.

```yaml
resources:
  base-allowlist:
    http: [github.com]

  my-tools:
    vars: [...]
    mounts: [...]
```

---

### `apply`

**Required:** Yes
**Type:** List of apply rules

Ordered list that maps workspace paths to resource sets. See [Apply Rules](#apply-rules) below.

```yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: frontend
    resources: [npm-tools]
```

---

## Resource Sets

Resource sets are defined under the `resources` key:

```yaml
resources:
  my-resource-set:
    vars: [...]
    mounts: [...]
    calls: [...]
    http: [...]
    ports: [...]
    root-commands: [...]
    options: {...}
```

### `vars`

**Type:** List of environment variable mappings

Maps host environment variables to container environment variables.

**Fields:**
- `source`: Name of host environment variable (required)
- `target`: Name of env var inside container (optional, defaults to `source`)

**Example:**
```yaml
resources:
  api-keys:
    vars:
      # When target is omitted, uses same name in container
      - source: OPENAI_API_KEY
      - source: DATABASE_URL

      # Only specify target when you want a different name
      - source: MY_HOST_API_KEY
        target: API_KEY
```

**Behavior:**
- Missing source env vars cause config loading to fail
- Values are never logged or exposed in config dumps
- If `target` is omitted, the variable keeps the same name in the container
- Only specify `target` when you need to rename the variable

---

### `mounts`

**Type:** List of bind mount specifications

Mounts host directories into the container.

**Fields:**
- `source`: Absolute path on the host (supports templates)
- `target`: Absolute path in the container (supports templates)
- `mode`: `ro` (read-only) or `rw` (read-write)

**Example:**
```yaml
resources:
  cache-mounts:
    mounts:
      # npm cache
      - source: ${{ env.HOME }}/.npm
        target: /home/${{ conf.TARGET_USER }}/.npm
        mode: rw

      # SSL certificates (read-only)
      - source: /etc/ssl/certs
        target: /etc/ssl/certs
        mode: ro

      # SSH keys (read-only)
      - source: ${{ env.HOME }}/.ssh
        target: /home/${{ conf.TARGET_USER }}/.ssh
        mode: ro
```

**Behavior:**
- Non-existent source directories are skipped with a warning
- Useful for optional mounts that may not exist on all machines
- Use `${{ conf.TARGET_USER }}` in target paths for user-specific directories

{{< callout type="warning" >}}
**Security:** Be cautious when mounting sensitive directories. Always use `mode: ro` for credentials and keys.
{{< /callout >}}

---

### `calls`

**Type:** List of remote call definitions

Defines host commands that can be invoked from inside the container.

**Fields:**
- `name`: Unique identifier for the call (required)
- `description`: Human-readable description (required)
- `command`: Absolute path to host command (required)
- `allowed-args`: Regex pattern to validate arguments (optional)

**Example:**
```yaml
resources:
  deployment:
    calls:
      - name: deploy-staging
        description: Deploy to staging environment
        command: /usr/local/bin/deploy.sh
        allowed-args: '^--env=staging --region=us-\w+-\d+$'

      - name: trigger-build
        description: Trigger CI build
        command: /usr/local/bin/trigger-build.sh
        # No allowed-args means no arguments permitted
```

**Behavior:**
- Call names must be unique within a single workspace path
- Arguments are validated against `allowed-args` regex before execution
- Missing `allowed-args` means the call accepts no arguments
- Calls are invoked with `shai-remote call <name> [args]` inside the container

{{< callout type="error" >}}
**Security:** Always use strict `allowed-args` patterns to prevent command injection.
{{< /callout >}}

See [Selective Elevation](/docs/concepts/selective-elevation) for more details.

---

### `http`

**Type:** List of hostnames

Defines which HTTP/HTTPS destinations are accessible from the container.

**Example:**
```yaml
resources:
  web-access:
    http:
      - github.com
      - api.openai.com
      - npmjs.org
      - pypi.org
```

**Behavior:**
- Only listed domains are accessible
- Subdomains are automatically included (e.g., `github.com` allows `api.github.com`)
- All other HTTP/HTTPS traffic is blocked
- Implemented via iptables, dnsmasq, and tinyproxy

{{< callout type="info" >}}
Don't include `http://` or `https://` prefixes. Just the hostname.
{{< /callout >}}

---

### `ports`

**Type:** List of host/port pairs

Allows access to specific TCP/UDP ports on specific hosts.

**Fields:**
- `host`: Hostname or IP address
- `port`: Port number

**Example:**
```yaml
resources:
  ssh-access:
    ports:
      - host: github.com
        port: 22

      - host: gitlab.com
        port: 22

      - host: database.internal
        port: 5432
```

**Use cases:**
- SSH access to git servers
- Database connections
- Custom TCP services

---

### `root-commands`

**Type:** List of shell commands

Commands to execute as root before switching to the target user.

**Example:**
```yaml
resources:
  docker-in-docker:
    root-commands:
      - "systemctl start docker"
      - "modprobe nbd"
    options:
      privileged: true
```

**Behavior:**
- Executed sequentially after container setup
- Run before user switch
- Any failure causes container to exit
- Only executed if container starts with root privileges

**Use cases:**
- Starting system services
- Loading kernel modules
- Configuring network interfaces

{{< callout type="warning" >}}
Root commands require careful consideration. They run with full root privileges.
{{< /callout >}}

---

### `options`

**Type:** Object with container-level options

**Fields:**
- `privileged`: Boolean (default: `false`)

**Example:**
```yaml
resources:
  gpu-access:
    options:
      privileged: true
```

**Behavior:**
- `privileged: true` runs the container in privileged mode
- Reduces container isolation
- Required for Docker-in-Docker, GPU access, etc.

{{< callout type="error" >}}
Use `privileged: true` sparingly! It significantly weakens security.
{{< /callout >}}

---

## Apply Rules

Apply rules are defined under the `apply` key:

```yaml
apply:
  - path: <workspace-relative-path>
    resources: [<resource-set-names>]
    image: <optional-image-override>
```

### `path`

**Required:** Yes
**Type:** String

Workspace-relative path that activates resource sets.

**Examples:**
```yaml
apply:
  - path: ./               # Matches everything
  - path: .                # Same as ./
  - path: frontend         # Matches frontend/**/*
  - path: backend/api      # Matches backend/api/**/*
```

**Behavior:**
- Paths are relative to workspace root
- Matches the path and all subdirectories
- `./` or `.` matches the entire workspace

---

### `resources`

**Required:** Yes
**Type:** List of strings

Names of resource sets to apply when the path matches.

**Example:**
```yaml
apply:
  - path: frontend
    resources: [base-allowlist, npm-registries, playwright]
```

**Behavior:**
- All listed resource sets are activated
- Resource sets must be defined in the `resources` section
- Call names within a path must be unique across all resource sets

---

### `image`

**Required:** No
**Type:** String

Optional container image override for this path.

**Example:**
```yaml
apply:
  - path: ./
    resources: [base-allowlist]
    # Uses default image from top-level config

  - path: infrastructure
    resources: [deployment-tools]
    image: ghcr.io/my-org/devops:latest
```

**Behavior:**
- Overrides the top-level `image` key for this path
- More specific paths take precedence
- Root path (`.` or `./`) cannot use image overrides
- CLI `--image` flag takes ultimate precedence

---

## Template Variables

The following template variables are available in config values:

### Environment Variables

```yaml
${{ env.NAME }}
```

References a host environment variable.

**Example:**
```yaml
image: ${{ env.DOCKER_IMAGE }}

resources:
  api-keys:
    vars:
      - source: OPENAI_API_KEY
```

**Behavior:**
- Missing env vars cause config loading to fail
- Available in all string fields

---

### User-Provided Variables

```yaml
${{ vars.NAME }}
```

References a variable provided via `--var` flag.

**Example:**
```yaml
# Config
image: ghcr.io/my-org/dev:${{ vars.TAG }}

# CLI
shai --var TAG=v1.2.3
```

**Behavior:**
- Missing vars cause config loading to fail
- Useful for parameterizing configs

---

### Configuration Variables

```yaml
${{ conf.TARGET_USER }}
${{ conf.WORKSPACE }}
```

References resolved configuration values.

**Available:**
- `TARGET_USER`: The resolved target user (default: `shai`)
- `WORKSPACE`: The resolved workspace path (default: `/src`)

**Example:**
```yaml
resources:
  cache:
    mounts:
      - source: ${{ env.HOME }}/.cache
        target: /home/${{ conf.TARGET_USER }}/.cache
        mode: rw
```

**Limitations:**
- Cannot be used in the `user` or `workspace` fields themselves
- Only available after those fields are resolved

---

## Complete Schema

```yaml
type: shai-sandbox
version: 1
image: <image-name>

# Optional
user: <username>
workspace: <path>

resources:
  <resource-set-name>:
    vars:
      - source: <VAR>           # Uses same name in container
      - source: <VAR2>
        target: <NEW_NAME>      # Renames in container

    mounts:
      - source: <host-path>
        target: <container-path>
        mode: ro|rw

    calls:
      - name: <call-name>
        description: <description>
        command: <host-command>
        allowed-args: <regex>

    http:
      - <hostname>

    ports:
      - host: <hostname>
        port: <port-number>

    root-commands:
      - <command>

    options:
      privileged: true|false

apply:
  - path: <workspace-path>
    resources: [<resource-set-names>]
    image: <optional-image-override>
```

## Validation

Shai validates the config when loading:

**Checks:**
- Required fields are present
- `type` is `shai-sandbox`
- `version` is `1`
- Resource set names are valid
- Apply rules reference existing resource sets
- Template variables are defined
- Paths are valid

**Behavior:**
- Invalid configs prevent Shai from starting
- Validation errors include helpful messages
- Use `shai --verbose` to debug config issues

## Next Steps

- See [Template Expansion](templates) for more on using variables
- Browse [Complete Example](example) for an annotated full config
- Check [Examples](/docs/examples) for real-world configurations
