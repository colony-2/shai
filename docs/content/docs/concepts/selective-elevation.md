---
title: Selective Elevation
weight: 4
---

**Selective Elevation** allows agents to perform specific host-side operations through controlled "remote calls" - commands that run on the host from inside the sandbox.

## The Problem

Containers provide isolation, but sometimes agents need to interact with the host:

- Flash firmware to a development board
- Deploy code to a server
- Run a build script that needs host resources
- Interact with host-side services
- Trigger verification workflows

Simply giving the container host access defeats the purpose of sandboxing. Selective elevation provides a middle ground: **curated, specific host operations**.

## How It Works

1. You define **remote calls** in your resource sets
2. Shai starts an MCP server on the host
3. Inside the container, agents invoke calls via `shai-remote`
4. The MCP server executes the command on the host
5. Output is returned to the agent

```
┌─────────────────────────────────────┐
│         Container (Sandbox)         │
│                                     │
│  Agent runs:                        │
│  $ shai-remote deploy-staging         │
│              │                      │
└──────────────┼──────────────────────┘
               │
               │ (MCP protocol)
               │
┌──────────────▼──────────────────────┐
│         Host (Your Machine)         │
│                                     │
│  MCP Server executes:               │
│  /usr/local/bin/deploy.sh           │
│                                     │
└─────────────────────────────────────┘
```

## Defining Remote Calls

Remote calls are defined in the `calls` section of a resource set:

```yaml
# .shai/config.yaml
resources:
  deployment:
    calls:
      - name: deploy-staging
        description: Deploy to staging environment
        command: /usr/local/bin/deploy.sh
        allowed-args: '^--env=staging$'

      - name: run-integration-tests
        description: Run integration tests on host
        command: /usr/local/bin/test-runner.sh
```

**Fields:**
- `name`: Unique identifier for the call (used with `shai-remote`)
- `description`: Human-readable description of what the call does
- `command`: Absolute path to the host command
- `allowed-args`: (Optional) Regex pattern to validate arguments

## Invoking Remote Calls

Inside the sandbox, use the `shai-remote` command:

```bash
# No arguments
shai-remote run-integration-tests

# With arguments (validated against allowed-args regex)
shai-remote deploy-staging --env=staging
```

The command runs on the **host**, and output is streamed back to the container.

{{< callout type="info" >}}
`shai-remote` is automatically available inside all Shai sandboxes. You don't need to install anything.
{{< /callout >}}

## Argument Filtering

The `allowed-args` field provides security through argument validation:

```yaml
calls:
  - name: flash-device
    description: Flash firmware to USB device
    command: /usr/local/bin/flash.sh
    allowed-args: '^--port=/dev/ttyUSB[0-9]+$'
```

**What this does:**
- Allows: `shai-remote flash-device --port=/dev/ttyUSB0` ✅
- Allows: `shai-remote flash-device --port=/dev/ttyUSB1` ✅
- Blocks: `shai-remote flash-device --port=/dev/sda` ❌
- Blocks: `shai-remote flash-device --rm -rf /` ❌

{{< callout type="warning" >}}
**Always use argument filtering** unless the call takes no arguments. This prevents agents from injecting malicious commands.
{{< /callout >}}

### Multiple Arguments

The regex validates the **entire argument string**:

```yaml
calls:
  - name: deploy
    description: Deploy to environment
    command: /usr/local/bin/deploy.sh
    allowed-args: '^--env=(staging|production) --region=us-\w+-\d+$'
```

Valid:
```bash
shai-remote deploy --env=staging --region=us-east-1
shai-remote deploy --env=production --region=us-west-2
```

Invalid:
```bash
shai-remote deploy --env=dev --region=us-east-1      # env=dev not allowed
shai-remote deploy --env=staging                     # missing --region
```

### No Argument Validation

If a command takes no arguments, omit `allowed-args`:

```yaml
calls:
  - name: trigger-build
    description: Trigger CI build
    command: /usr/local/bin/trigger-build.sh
    # No allowed-args means no arguments are permitted
```

## Use Cases

### Firmware Flashing

```yaml
resources:
  embedded-dev:
    calls:
      - name: flash-firmware
        description: Flash compiled firmware to device
        command: /usr/local/bin/flash.sh
        allowed-args: '^--device=/dev/ttyUSB[0-9]+ --binary=/tmp/firmware-\w+\.bin$'
```

Agent workflow:
1. Compile firmware inside sandbox
2. Copy binary to `/tmp/firmware-xyz.bin`
3. `shai-remote flash-firmware --device=/dev/ttyUSB0 --binary=/tmp/firmware-xyz.bin`

### Deployment Verification

```yaml
resources:
  deployment:
    calls:
      - name: verify-deployment
        description: Verify deployment succeeded
        command: /usr/local/bin/verify.sh
        allowed-args: '^--service=[\w-]+ --env=(staging|production)$'
```

Agent workflow:
1. Deploy code (via other means)
2. `shai-remote verify-deployment --service=api --env=staging`
3. Check exit code to confirm deployment

### Secret Fetching

```yaml
resources:
  secret-access:
    calls:
      - name: fetch-secrets
        description: Fetch secrets from host vault
        command: /usr/local/bin/fetch-secrets.sh
        allowed-args: '^--env=(dev|staging|production)$'
```

{{< callout type="warning" >}}
Be careful with secret-fetching calls. Ensure the host script validates the caller and logs access.
{{< /callout >}}

### Build Triggering

```yaml
resources:
  ci-integration:
    calls:
      - name: trigger-ci
        description: Trigger CI pipeline
        command: /usr/local/bin/trigger-ci.sh
        allowed-args: '^--branch=[\w/-]+ --pipeline=[\w-]+$'
```

### Database Operations

```yaml
resources:
  db-admin:
    calls:
      - name: create-migration
        description: Create database migration
        command: /usr/local/bin/create-migration.sh
        allowed-args: '^--name=[\w_]+$'

      - name: rollback-migration
        description: Rollback last migration
        command: /usr/local/bin/rollback.sh
        allowed-args: '^--steps=[1-9]$'
```

## Security Considerations

### ✅ Do

- **Validate arguments rigorously** with `allowed-args`
- Use absolute paths for commands
- Keep host scripts simple and auditable
- Log all call invocations on the host
- Treat calls as security-sensitive boundaries
- Use calls sparingly - prefer in-container operations

### ❌ Don't

- Allow arbitrary arguments without regex validation
- Give calls shell access (`command: /bin/bash`)
- Allow file path arguments without strict validation
- Trust user input without validation
- Use calls for operations that could run in-container

### Argument Injection Risks

**Bad:**
```yaml
calls:
  - name: deploy
    command: /bin/bash -c "deploy.sh"
    allowed-args: '.*'    # ❌ Allows anything!
```

**Good:**
```yaml
calls:
  - name: deploy
    command: /usr/local/bin/deploy.sh
    allowed-args: '^--env=(staging|production)$'    # ✅ Strict validation
```

## Debugging Calls

### List Available Calls

Inside the sandbox:

```bash
shai-remote --list
```

Shows all calls available in your current resource sets.

### Test Calls

Manually test calls to verify behavior:

```bash
shai -rw . --resource-set deployment
# Inside sandbox:
shai-remote deploy-staging --env=staging
```

### Host Script Output

Host script stdout/stderr is sent back to the container. Use this for debugging:

```bash
#!/bin/bash
# /usr/local/bin/deploy.sh

echo "Starting deployment..." >&2
echo "Arguments: $@" >&2

# Do deployment work...

echo "Deployment complete!"
```

## Call Uniqueness

Within a single workspace path, all call names must be unique:

```yaml
resources:
  set-a:
    calls:
      - name: deploy
        command: /usr/local/bin/deploy-a.sh

  set-b:
    calls:
      - name: deploy        # ❌ Error if both sets apply to same path
        command: /usr/local/bin/deploy-b.sh
```

If you need multiple deployment calls, use distinct names:

```yaml
resources:
  staging-deploy:
    calls:
      - name: deploy-staging
        command: /usr/local/bin/deploy-staging.sh

  production-deploy:
    calls:
      - name: deploy-production
        command: /usr/local/bin/deploy-production.sh
```

## Alternatives to Remote Calls

Before using remote calls, consider if the operation can happen in-container:

| Goal | In-Container Alternative |
|------|--------------------------|
| Access secrets | Mount secret files read-only |
| Run tests | Install test tools in the image |
| Build artifacts | Use multi-stage builds |
| Network access | Add to HTTP allowlist |

Remote calls are best for operations that **must** interact with host-specific resources (hardware devices, host-only services, etc.).

## Next Steps

- See [Resource Sets](../resource-sets) for more details on defining calls
- Learn about [Security](/docs/security) best practices
- Browse [Examples](/docs/examples) for real-world call patterns
