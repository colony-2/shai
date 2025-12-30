---
title: Template Expansion
weight: 2
---

Shai config files support template variables for dynamic configuration.

## Overview

Templates allow you to:
- Reference environment variables
- Parameterize configs with CLI flags
- Use resolved configuration values
- Avoid hardcoding secrets

## Template Syntax

Templates use the `${{ ... }}` syntax:

```yaml
${{ env.VARIABLE_NAME }}    # Host environment variable
${{ vars.VARIABLE_NAME }}   # CLI-provided variable
${{ conf.VARIABLE_NAME }}   # Resolved configuration value
```

## Environment Variables

Reference host environment variables with `${{ env.NAME }}`.

### Example: Dynamic Image Selection

```yaml
image: ${{ env.DEV_IMAGE }}
```

```bash
export DEV_IMAGE=ghcr.io/my-org/dev:latest
shai -rw src
```

### Example: API Keys

```yaml
resources:
  api-access:
    vars:
      # target is optional - omit when name stays the same
      - source: OPENAI_API_KEY
      - source: ANTHROPIC_API_KEY
```

### Example: User-Specific Paths

```yaml
resources:
  cache-mounts:
    mounts:
      - source: ${{ env.HOME }}/.cache/models
        target: /home/shai/.cache/models
        mode: rw
```

### Behavior

- **Missing variables cause failure:** If `OPENAI_API_KEY` is not set, config loading fails
- **This is intentional:** You catch missing credentials early instead of at runtime
- **Use for secrets:** Never hardcode credentials in config files

{{< callout type="info" >}}
**Best Practice:** Always use `${{ env.* }}` for credentials instead of hardcoding them.
{{< /callout >}}

## User Variables

Provide variables via the `--var` flag with `${{ vars.NAME }}`.

### Example: Parameterized Image Tags

```yaml
image: ghcr.io/my-org/dev:${{ vars.TAG }}
```

```bash
shai --var TAG=v1.2.3 -rw src
```

### Example: Branch-Specific Configuration

```yaml
resources:
  deployment:
    calls:
      - name: deploy
        description: Deploy to environment
        command: /usr/local/bin/deploy.sh
        allowed-args: '^--branch=${{ vars.BRANCH }}$'
```

```bash
shai --var BRANCH=main -rw infrastructure
```

### Multiple Variables

```bash
shai --var TAG=v2.0.0 --var ENV=staging --var REGION=us-east-1
```

```yaml
image: ghcr.io/my-org/app:${{ vars.TAG }}

resources:
  deployment:
    vars:
      - source: AWS_CREDENTIALS
    calls:
      - name: deploy
        description: Deploy application
        command: /usr/local/bin/deploy.sh
        allowed-args: '^--env=${{ vars.ENV }} --region=${{ vars.REGION }}$'
```

### Behavior

- **Syntax:** `--var KEY=VALUE`
- **Multiple vars:** Repeat `--var` flag
- **Missing vars:** Cause config loading to fail

## Configuration Variables

Reference resolved configuration values with `${{ conf.NAME }}`.

### Available Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TARGET_USER` | The resolved target user | `shai` |
| `WORKSPACE` | The resolved workspace path | `/src` |

### Example: User Home Directory

```yaml
resources:
  cache:
    mounts:
      - source: ${{ env.HOME }}/.npm
        target: /home/${{ conf.TARGET_USER }}/.npm
        mode: rw
```

With default user (`shai`):
- Target becomes: `/home/shai/.npm`

With custom user:
```bash
shai --user developer
```
- Target becomes: `/home/developer/.npm`

### Example: Workspace-Relative Paths

```yaml
resources:
  tools:
    mounts:
      - source: /host/tools
        target: ${{ conf.WORKSPACE }}/tools
        mode: ro
```

With default workspace:
- Target becomes: `/src/tools`

### Restrictions

{{< callout type="warning" >}}
You **cannot** use `${{ conf.* }}` in the `user` or `workspace` fields themselves:

```yaml
# ❌ This doesn't work
user: ${{ conf.TARGET_USER }}
workspace: ${{ conf.WORKSPACE }}
```

These fields are used to **build** the `conf` variables, so they can't reference them.
{{< /callout >}}

## Where Templates Work

Templates can be used in most string fields:

### ✅ Supported

```yaml
# Top-level
image: ${{ env.IMAGE }}
user: ${{ env.USER }}
workspace: ${{ env.WORKSPACE }}

# Resource sets - vars
vars:
  - source: API_KEY              # Uses same name
  - source: HOST_KEY
    target: CONTAINER_KEY        # Rename if needed

# Resource sets - mounts
mounts:
  - source: ${{ env.HOME }}/.cache
    target: /home/${{ conf.TARGET_USER }}/.cache
    mode: rw

# Resource sets - calls
calls:
  - name: deploy
    description: Deploy app
    command: ${{ env.DEPLOY_SCRIPT }}
    allowed-args: '^--env=${{ vars.ENV }}$'

# Apply rules
apply:
  - path: ./
    image: ${{ env.BASE_IMAGE }}
```

### ❌ Not Supported

Templates don't work in:
- Field names/keys
- The `type` field
- The `version` field
- Boolean values
- Numeric values

## Combining Templates

You can combine multiple template types:

```yaml
resources:
  custom:
    mounts:
      # Combines env and conf variables
      - source: ${{ env.HOME }}/.config/app
        target: /home/${{ conf.TARGET_USER }}/.config/app
        mode: rw

    calls:
      # Combines env and vars
      - name: deploy
        description: Deploy to ${{ vars.ENV }}
        command: ${{ env.SCRIPTS_DIR }}/deploy.sh
        allowed-args: '^--env=${{ vars.ENV }}$'
```

## Error Handling

### Missing Environment Variable

**Config:**
```yaml
image: ${{ env.MISSING_VAR }}
```

**Error:**
```
Error: Failed to load config
Cause: Template variable not found: env.MISSING_VAR
```

### Missing User Variable

**Config:**
```yaml
image: ghcr.io/app:${{ vars.TAG }}
```

**Command:**
```bash
shai -rw src
# Missing --var TAG=...
```

**Error:**
```
Error: Failed to load config
Cause: Template variable not found: vars.TAG
```

### Solution

Ensure all referenced variables are defined:

```bash
export MISSING_VAR=value
shai --var TAG=v1.0.0 -rw src
```

## Best Practices

### ✅ Do

**Use env vars for secrets:**
```yaml
vars:
  - source: DATABASE_PASSWORD
```

**Use vars for parameterization:**
```yaml
image: ghcr.io/app:${{ vars.VERSION }}
```

**Use conf vars for user-specific paths:**
```yaml
target: /home/${{ conf.TARGET_USER }}/.config
```

**Fail fast with required vars:**
```yaml
# This will fail immediately if API_KEY is missing
vars:
  - source: API_KEY
```

### ❌ Don't

**Don't hardcode secrets:**
```yaml
# ❌ Bad - secret in config file
vars:
  - source: "sk-1234567890abcdef"
    target: API_KEY

# ✅ Good - secret from environment
vars:
  - source: API_KEY
```

**Don't use templates where they're not supported:**
```yaml
# ❌ Bad - templates in boolean values
options:
  privileged: ${{ env.PRIVILEGED }}

# ✅ Good - use conditional config files instead
```

**Don't rely on default values:**
```yaml
# ❌ Bad - assumes HOME is set (it usually is, but be explicit)
# ✅ Good - document required env vars in README
```

## Common Patterns

### Multi-Environment Config

```yaml
resources:
  deployment:
    vars:
      # Different credentials per environment
      - source: AWS_ACCESS_KEY_ID
      - source: AWS_SECRET_ACCESS_KEY

    calls:
      - name: deploy
        description: Deploy to ${{ vars.ENV }}
        command: /usr/local/bin/deploy.sh
        allowed-args: '^--env=${{ vars.ENV }}$'
```

**Usage:**
```bash
# Staging
shai --var ENV=staging -rw infrastructure

# Production
shai --var ENV=production -rw infrastructure
```

### User-Specific Caching

```yaml
resources:
  caches:
    mounts:
      # npm
      - source: ${{ env.HOME }}/.npm
        target: /home/${{ conf.TARGET_USER }}/.npm
        mode: rw

      # cargo
      - source: ${{ env.HOME }}/.cargo
        target: /home/${{ conf.TARGET_USER }}/.cargo
        mode: rw

      # pip
      - source: ${{ env.HOME }}/.cache/pip
        target: /home/${{ conf.TARGET_USER }}/.cache/pip
        mode: rw
```

### Dynamic Image Selection

```yaml
# Development
image: ${{ env.DEV_IMAGE }}

# Or with fallback via shell default
# export DEV_IMAGE=${DEV_IMAGE:-ghcr.io/colony-2/shai-mega}
```

## Debugging Templates

Use `--verbose` to see expanded template values:

```bash
shai -rw src --verbose
```

Output shows:
- Raw config before expansion
- Expanded config after template processing
- Which variables were used

## Next Steps

- See [Schema Reference](schema) for all available fields
- Browse [Complete Example](example) for a full annotated config
- Check [Examples](/docs/examples) for real-world configurations
