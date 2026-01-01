# shai - a sandboxing shell for ai coding agents

[![npm version](https://img.shields.io/npm/v/@colony2/shai.svg)](https://www.npmjs.com/package/@colony2/shai)
[![Homebrew](https://img.shields.io/badge/homebrew-shai-blue.svg)](https://github.com/colony-2/homebrew-tap)
[![Docker Image shai-base](https://img.shields.io/badge/ghcr.io-shai--base-blue.svg)](https://github.com/colony-2/shai/pkgs/container/shai-base)
[![Docker Image shai-mega](https://img.shields.io/badge/ghcr.io-shai--mega-blue.svg)](https://github.com/colony-2/shai/pkgs/container/shai-mega)

Shai (pronounced "shy") is a sandboxing tool for running CLI-based AI agents inside containers. At it's core, running `shai` will place your terminal inside a container mounted at the current path. This container mounts a read-only copy of your current path at /src as a non-root user and restricts network access to a select list of http and https destinations. All other network traffic is blocked. This is the perfect sandbox for running CLI agents like [Codex](https://github.com/openai/codex), [Claude Code](https://github.com/anthropics/claude-code), [Gemini CLI](https://github.com/google-gemini/gemini-cli), [Cline](https://github.com/cline/cline), [OpenHands](https://github.com/OpenHands/OpenHands), etc.

##  Quick Start
1. Install shai:
2. 
   ```bash
   npm install -g @colony2/shai
   ```

   or

   ```bash
   brew install --cask colony-2/tap/shai
   ```
 
2. Run Shai from your workspace:
   ```bash
   # minimal invocation – read-only workspace sandbox
   shai

   # mark specific directories as writable
   shai -rw app/component1

   # run a command after bootstrap
   shai -rw app/component1 -- codex --dangerously-bypass-approvals-and-sandbox

   # inspect generated bootstrap script and resource decisions
   shai -rw app/component1 src --verbose

   # override the Docker image explicitly
   shai -rw app/component1 --image ghcr.io/example/dev:latest
   ```

### Useful flags
- `--read-write, -rw <path>` (repeatable) – declare relative paths that receive writable overlays inside `/src`; omit to keep the workspace entirely read-only. Paths must exist or an error will be raised.
- `--resource-set, -rs <name>` – opt into additional resource sets beyond the config's apply rules.
- `--image, -i <image>` – override the container image; this takes precedence over apply-rule overrides.
- `--user, -u <user>` – override the target container user; takes precedence over config file.
- `--privileged` – run the container in privileged mode (can also be set per-resource-set).
- `--var, -v KEY=value` – provide template variables consumed by `${{ vars.KEY }}` expressions.
- `--verbose, -V` – dump bootstrap details.
- `--no-tty, -T` – disable TTY allocation for the post-setup command (structured log mode).

If you pass `-- command ...`, those arguments become the `PostSetupExec` inside the container. Without a command, Shai switches to the configured user and drops you into an interactive login shell.

## Cellular Software Development & Target Paths
Shai is built around the concept of cellular software development. In this model, agents are given constrained access to individual components as opposed to cross-repo access. They can consume and understand related code but must limit changes to individual components. When Shai is started, a user specifies the specific subdirectory (or subdirectories) that the session will be limited to. This is done via the -rw (or --read-write) flag. For example, if you wanted to give an agent write access to the `agents/research` directory, you would run: `shai -rw agents/research`. This mounts that subdirectory inside the container at `/src/agents/research` while mounting the rest of the workspace read-only.

## Resource Sets
It is often the case that coding agents should have access to different sets of resources depending on context. For example, a Rust application development agent shouldn't have access to production deployment credentials while a Pulumi deployment module built using shouldn't have access to a Rust compiler or arbitrary Github repositories. These collections of resources are called Resource Sets in `shai`. You can define an arbitrary number of named resource sets. Resource sets contain each of the following:
- Valid HTTP/HTTPS destinations
- Host mounts
- IP Ports
- Environment variables
- Remote Calls
- Root Commands (optional commands run as root before user switch)

## Apply Rules
Shai allows you to define a set of rules that define which resource sets and container images are applied to which target paths. This allows you to define a set of default resource sets that apply to all subdirectories, and then enhance them for specific subtrees. 

## Selective Elevation - Calls
Sometimes, agents need to perform operations that are outside the scope of their containerized environment. For example, an agent working on embedded firmware may need to be able to flash that code to a specific host-mounted development board. Shai allows you to define specific host-side commands that can be called from inside the container. These remote calls are defined in the `calls` section of a resource set.

## `.shai/config.yaml` Reference
### Generating a default config
You can generate a default config file (optional):
```bash
shai generate
```
This creates `.shai/config.yaml` with sensible defaults. You can also rely on the embedded default config if no custom configuration is needed.

Shai automatically loads `<workspace>/.shai/config.yaml` unless `--config` overrides it. Missing configs fall back to the [embedded default config](https://github.com/colony-2/shai/blob/main/internal/shai/runtime/config/shai.default.yaml), which primarily enables a permissive HTTP allow-list and open-source registries.

### Top-level keys
| Key | Required | Description |
| --- | --- | --- |
| `type` | yes | Must be `shai-sandbox`. Used for schema validation.
| `version` | yes | Must be `1`. Future releases may bump this.
| `image` | yes | Base container image. Can use templates.
| `user` | no | Container user Shai switches to before running your command. Defaults to `shai`. Can be overridden with `--user` flag.
| `workspace` | no | Absolute path of the repository inside the container. Defaults to `/src`.
| `resources` | yes | Map of resource-set definitions (see below).
| `apply` | yes | Ordered list that maps workspace paths to resource sets and optional image overrides.

### Resource sets
```yaml
resources:
  my-tools:
    vars:
      - source: OPENAI_API_KEY
    mounts:
      - source: ${{ env.HOME }}/.cache/model
        target: /home/${{ conf.TARGET_USER }}/.cache/model
        mode: rw
    calls:
      - name: fetch-secrets
        description: Fetch secrets from the host vault
        command: /usr/local/bin/fetch-secrets.sh
        allowed-args: '^--env=\w+$'
    http:
      - api.openai.com
      - github.com
    ports:
      - host: github.com
        port: 22
    root-commands:
      - "systemctl start docker"
      - "modprobe nbd"
    options:
      privileged: false
```
- `vars` – Maps host environment variable names (`source`) to container variables (`target`). The `source` field should be the plain environment variable name (e.g., `OPENAI_API_KEY`), not a template expression. The `target` field is optional; if omitted, the variable keeps the same name in the container. Missing env variables cause load failures.
- `mounts` – Bind mount host paths into the container. `mode` defaults to `ro`; valid values are `ro` or `rw`. Non-existent source directories are skipped with a warning at startup. Use `${{ conf.TARGET_USER }}` in target paths to reference the configured user.
- `calls` – Expose curated host commands inside the sandbox. Names must be unique per path, `command` is executed on the host, and `allowed-args` (optional) is a regex that filters arguments forwarded from inside the container.
- `http` – Hostnames the sandbox is allowed to reach. Use this to tighten egress beyond the defaults.
- `ports` – Explicit host/port pairs that Shai proxies so agents can reach ssh servers or custom endpoints.
- `root-commands` – (Optional) Shell commands to execute in the root user context before switching to the target user. These commands run after all container setup is complete (network filtering, user creation, etc.) but before the user switch. Commands are executed sequentially, and any failure will cause the container to exit with an error. Useful for starting services (e.g., `systemctl start docker`) or loading kernel modules (e.g., `modprobe nbd`) that require root privileges. Root commands are only executed when the container is running with root privileges; if the container starts as a non-root user, these commands are skipped.
- `options` – Optional settings for this resource set:
  - `privileged` – (defaults to `false`) When `true`, enables privileged mode for the container when this resource set is active. Use with caution as this reduces isolation.

### Apply rules
```yaml
apply:
  - path: ./
    resources: [base-allowlist]
  - path: agents/research
    resources: [my-tools, gpu-mount]
    image: ghcr.io/my-org/gpu:latest
```
- `path` – Workspace-relative path that activates resource sets for itself and its subdirectories. `./` (or `.`) matches everything.
- `resources` – Names of resource sets applied by this rule. Within a single path, resource-set calls must define unique `name` values.
- `image` – Optional override that replaces the base image when Shai is invoked from the matching subtree. The root (`.`) rule cannot override the image; use more specific paths.

Rules are evaluated top to bottom. When resolving a workspace path, Shai aggregates all matching resource sets (deduplicated) and selects the most specific image override. CLI `--resource-set` flags append to the resolved list.

### Template expansion
Any string field can embed:
- `${{ env.NAME }}` – host environment variable
- `${{ vars.NAME }}` – provided via `--var` or the Go API
- `${{ conf.TARGET_USER }}` – the resolved target user (defaults to `shai` or overridden via `--user`)
- `${{ conf.WORKSPACE }}` – the resolved workspace path (defaults to `/src`)

Missing references cause config loading to fail so you catch mistakes early.

**Note:** The `user` and `workspace` fields themselves can only use `env` and `vars` templates, not `conf` templates (since `conf` is built from these values).

### Example
```yaml
# .shai/config.yaml
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-mega
# user and workspace are optional - defaults are user: shai, workspace: /src
resources:
  base-allowlist:
    http: [openai.com, api.openai.com]
  agent-dev:
    vars:
      - source: OPENAI_API_KEY
    mounts:
      - source: ${{ env.HOME }}/.cache/agents
        target: /home/${{ conf.TARGET_USER }}/.cache/agents
        mode: rw
    calls:
      - name: refresh-plan
        description: Refresh cached plans on the host
        command: ${{ env.REPO_ROOT }}/scripts/refresh-plan.sh
    ports:
      - host: github.com
        port: 22
apply:
  - path: ./
    resources: [base-allowlist]
  - path: agents
    resources: [agent-dev]
```

## How it works
Shai builds on top of Docker and Docker-compatible daemons. Shai starts an ephemeral container with a generated name in the format `shai-<random>`. In this container it sets an entrypoint of a bootstrap script mounted by shai. This bootstrap script sets up additional sandboxing beyond what the base container provides including defining firewalls rules via iptables and setting up a reverse proxy and dns server to restrict external access. Shai also starts a host-side MCP server that is accessible via injected credentials in the container, allowing container access to the remote calls defined in the config. Once the sandbox environment is setup, Shai exec's as the provided user command as a non-privileged user.

### Security Features
- **Config file protection**: When the workspace root (`.`) is mounted as read-write, Shai automatically remounts `.shai/config.yaml` as read-only to prevent unintended sandbox escapes through config modification.
- **iptables logging**: Network firewall rules are logged to `/var/log/shai/iptables.out` after setup, allowing non-root users to inspect the active network restrictions.
- **Container isolation**: Containers run as auto-remove ephemeral instances with network filtering, limited capabilities, and read-only workspace mounts by default.

## Docker Images
Shai can work with any Docker image that follows Linux standards and has the required system utilities installed. The bootstrap process automatically starts `supervisord` and loads any additional service configurations defined in `/etc/supervisor/conf.d/*.conf`, making it easy to extend containers with custom background services.

### Requirements
A compatible Docker image must include:
- **supervisord** – Process supervisor for managing background services
- **dnsmasq** – DNS server for domain filtering
- **iptables** – Firewall for network egress control
- **tinyproxy** – HTTP/HTTPS proxy for allow-listed traffic
- **Core utilities** – bash, coreutils, iproute2, iputils-ping, jq, net-tools, passwd, procps, sed, util-linux

### shai-base Image
The `ghcr.io/colony-2/shai-base:latest` image is a minimal Debian-based image containing only the essential packages required for shai sandboxing to function. This image is ideal for:
- Building custom development images with specific language runtimes
- Fast test execution (smaller image size means faster pulls in CI/CD)
- Scenarios where you want full control over installed tools

**Included in shai-base:**
- **System Utilities:** bash, ca-certificates, coreutils, curl, iproute2, iputils-ping, jq, net-tools, passwd, procps, sed, util-linux
- **Sandboxing Tools:** supervisor, dnsmasq, iptables, tinyproxy

The shai-base image is based on `debian:bookworm-slim` and serves as the foundation for the shai-mega image.

### shai-mega Image
Shai is preconfigured to use the `ghcr.io/colony-2/shai-mega` image, which provides a comprehensive development environment based on the latest Debian stable (bookworm-slim). This is a kitchen-sink image containing common development tools and language runtimes, so the initial download may take a minute, but it enables a wide range of development patterns without requiring custom images.

**Included in shai-mega:**
- **Languages & Runtimes:** Go (latest), Rust (stable), Node.js (latest), Python 3, Java (default JDK), C/C++ (GCC/Clang)
- **Package Managers:** npm, yarn, pnpm, pip, cargo
- **AI CLI Tools:** OpenAI Codex, Google Gemini CLI, Anthropic Claude Code, Moonrepo
- **Browser Automation:** Playwright with Chromium
- **Development Tools:** git, jq, curl, wget, bash-completion, vim, nano, htop, tree, rsync, ssh
- **Build Tools:** build-essential, pkg-config, make
- **System Utilities:** supervisor, iptables, tinyproxy, dnsmasq, zsh

All language toolchains are installed system-wide at `/usr/local` with appropriate PATH configuration for all users.

## Inspiration & Prior Art
The first version of Shai was built on top of [devcontainers](https://containers.dev) (The original underlying library can be [seen here](https://github.com/colony-2/devcontainer-go)). Devcontainers are great and were a good place for Shai to start. However, over time, the deisgn goal differences between Shai and Devcontainers became challenging and thus we ultimately decided to define an alternative configuration:
- the devcontainer spec is not designed for segmented configuration (e.g. subdirectory `a` gets different resources than subdirectory `b`).
- devcontainers are expected to be longer-lived. Features assumes this (features can take 30s to many minutes to install). Using features in Shai meant every new session had a large startup wait.
- there are a lot of features in devcontainers. shai only needed a small subset of them.
- devcontainers doesn't really have any concept of sandboxing controls for things like firewalling, etc. you can define these things in a feature or in the container image, but they're basically diy.
- the devcontainers tools are not really built for throw-away ephemeral containers
- devcontainers don't actively mitigate again things like inclusion of credentials in configuration. (Shai does by doing things like only allowing mapping of ENV variables as opposed to declaration of them directly in config.)

Shai has no desire to replace devcontainers. They are focused on two different use cases.


## Embedding the Go API
Advanced users can embed Shai inside Go binaries using `pkg/shai`:
```go
import (
    "context"
    shai "github.com/colony-2/shai/pkg/shai"
)

func runAgent(ctx context.Context, repoRoot string) error {
    cfg, err := shai.LoadSandboxConfig(repoRoot,
        shai.WithReadWritePaths([]string{"agents/cache"}),
        shai.WithResourceSets([]string{"gpu"}),
        shai.WithTemplateVars(map[string]string{"BRANCH": "main"}),
    )
    if err != nil {
        return err
    }
    cfg.PostSetupExec = &shai.SandboxExec{
        Command: []string{"python", "-m", "agent"},
        Workdir: "/src/agents",
        UseTTY: false,
    }
    sandbox, err := shai.NewSandbox(cfg)
    if err != nil {
        return err
    }
    defer sandbox.Close()
    return sandbox.Run(ctx)
}
```
Key types:
- `SandboxConfig` – Describes the workspace, config path, read/write overlays, selected resource sets, template variables, optional exec command, log writers, verbosity, graceful stop timeout, and image overrides.
- `SandboxExec` – Encapsulates the post-setup command (`Command`, env map, `Workdir`, `UseTTY`).
- `Sandbox` – Interface with `Run`, `Start`, and `Close`. `Start` returns a `SandboxSession` with `ContainerID`, `Wait`, `Stop`, and `Close` helpers for supervising long-running jobs.

Use the Go API when you need to orchestrate multiple sandboxes, integrate with supervisors, or reuse Shai as the execution backend inside unit/integration tests.
