---
title: Quick Start
weight: 1
---

Get up and running with Shai in minutes.

## Installation

Choose your preferred installation method:

{{< tabs >}}

  {{< tab name="npm" >}}
  ### npm

  ```bash
  npm install -g @colony2/shai
  ```
  {{< /tab >}}

  {{< tab name="Homebrew" >}}
  ### Homebrew

  ```bash
  brew install --cask colony-2/tap/shai
  ```
  {{< /tab >}}

{{< /tabs >}}

### Verify Installation

```bash
shai --version
```

You should see the version number printed to the console.

## Your First Sandbox

Let's start a basic sandbox to understand how Shai works.

### 1. Navigate to Your Project

```bash
cd ~/projects/my-app
```

### 2. Start Shai

```bash
shai
```

This opens a sandboxed shell with:
- Your workspace mounted **read-only** at `/src`
- Network access restricted to common package registries and APIs
- Running as non-root user `shai`

### 3. Explore the Sandbox

Inside the sandbox, try these commands:

```bash
# View your code (read-only)
ls /src

# Try to create a file (this will fail)
echo "test" > /src/file.txt
# Output: cannot create /src/file.txt: Read-only file system

# Check network restrictions
curl https://github.com
# Works! (GitHub is in the default allowlist)

# Exit the sandbox
exit
```

{{< callout type="info" >}}
**By default, the entire workspace is read-only.** This prevents unintended modifications to your code.
{{< /callout >}}

## Making Directories Writable

To allow modifications to specific directories, use the `-rw` (or `--read-write`) flag:

### Single Directory

```bash
shai -rw src/components
```

This mounts `src/components` as writable while keeping the rest of your workspace read-only.

### Multiple Directories

```bash
shai -rw src/components -rw tests/unit
```

You can specify multiple `-rw` flags to make multiple directories writable.

{{< callout type="warning" >}}
**Paths must exist!** Shai will error if you specify a path that doesn't exist in your workspace.
{{< /callout >}}

## Running AI Agents

The real power of Shai comes from running AI agents inside the sandbox.

### Launch an Agent After Bootstrap

```bash
shai -rw src/auth -- claude-code --dangerously-bypass-approvals-and-sandbox
```

**What this does:**
1. Creates a sandbox with `src/auth` writable
2. After sandbox setup completes, launches `claude-code`
3. The agent runs inside the controlled environment

### Without a Command

If you don't specify a command after `--`, Shai drops you into an interactive shell where you can manually run agents:

```bash
shai -rw src/components
# Now inside sandbox
claude-code
```

{{< callout type="info" >}}
**Why `--dangerously-bypass-approvals-and-sandbox`?**

Many AI agents have their own sandboxing. When running inside Shai, you can disable the agent's built-in sandbox since Shai provides superior isolation that is consistent across agents.
{{< /callout >}}

## Common Workflows

### Frontend Development

```bash
shai -rw src -rw public -- claude-code
```

### Backend Development

```bash
shai -rw internal/api -rw tests -- codex
```

### Running Tests

```bash
shai -- npm test
```

Tests can read your code but won't modify it.

## Inspecting the Sandbox

Want to see what Shai is doing? Use verbose mode:

```bash
shai -rw src/components --verbose
```

This shows:
- Bootstrap script details
- Resource decisions (which resource sets are applied)
- Network firewall rules
- Mount points

## Next Steps

Now that you've got Shai running, dive deeper:

{{< cards >}}
  {{< card link="../concepts" title="Core Concepts" icon="book-open" subtitle="Learn about cellular development, resource sets, and apply rules" >}}
  {{< card link="../configuration" title="Configuration" icon="cog" subtitle="Create a custom .shai/config.yaml for your project" >}}
  {{< card link="../examples" title="Examples" icon="document-text" subtitle="See real-world configuration examples" >}}
{{< /cards >}}

## Troubleshooting

### Docker Not Running

```
Error: Cannot connect to Docker daemon
```

**Solution:** Start Docker Desktop or the Docker daemon.

### Permission Denied

```
Error: permission denied while trying to connect to the Docker daemon socket
```

**Solution:** Add your user to the `docker` group:

```bash
sudo usermod -aG docker $USER
```

Then log out and back in.

### More Help

See the [Troubleshooting guide](/docs/troubleshooting) for more common issues and solutions.
