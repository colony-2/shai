# shai-mega

A comprehensive Debian-based development environment with modern programming languages, build tools, and AI CLI tools.

## What's Included

### Programming Languages & Runtimes
- **Go** - Latest stable version (overridable)
- **Rust** - Latest stable version (overridable)
- **Node.js** - Latest stable version (overridable)
- **Python 3** - From Debian repositories with pip, venv, and dev tools
- **Java** - OpenJDK from Debian repositories
- **C/C++** - GCC, Clang, and build essentials

### Package Managers & Build Tools
- **npm, yarn, pnpm** - Latest versions from npm registry
- **cargo** - Rust package manager
- **pip** - Python package manager
- **build-essential** - GCC, make, and core build tools

### AI & Development CLIs
- **@openai/codex** - OpenAI Codex CLI
- **@google/gemini-cli** - Google Gemini CLI
- **@anthropic-ai/claude-code** - Anthropic Claude Code CLI
- **@moonrepo/cli** - Moon build system
- **playwright** - Browser automation (includes Chromium with deps)

### System Tools & Utilities
- **Shells**: bash, zsh (available, not configured as default)
- **Version Control**: git
- **Utilities**: jq, curl, wget, rsync, tree, htop, ncdu, strace
- **Networking**: iproute2, net-tools, openssh-client, sshpass, tinyproxy, dnsmasq
- **Editors**: nano, vim-tiny
- **Process Management**: supervisor, inotify-tools

## Image Details

- **Base Image**: debian:bookworm-slim
- **Working Directory**: `/src`
- **Architectures**: amd64 (x86_64), arm64 (aarch64)
- **Configuration**: Uses Linux standard patterns (`/etc/profile.d`, `/etc/supervisor/conf.d`)

## Version Management

This image uses a **multi-stage build** to automatically fetch the latest versions of all dependencies at build time. When you build the image without specifying versions, it will use the latest stable releases.

### Automatic Latest Versions

By default, the following are fetched automatically:
- **Languages**: Go, Rust, Node.js
- **npm packages**: npm, yarn, pnpm, codex, gemini-cli, claude-code, moon, playwright

### Overriding Versions

You can pin specific versions using build arguments:

```bash
# Pin Go version
docker build --build-arg GO_VERSION=1.24.0 .

# Pin Rust version
docker build --build-arg RUST_VERSION=1.83.0 .

# Pin Node.js version
docker build --build-arg NODE_VERSION=22.11.0 .

# Pin multiple versions
docker build \
  --build-arg GO_VERSION=1.24.0 \
  --build-arg RUST_VERSION=1.83.0 \
  --build-arg NODE_VERSION=22.11.0 \
  .
```

### Available Build Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `GO_VERSION` | Go version (e.g., `1.24.0`) | Latest from go.dev |
| `RUST_VERSION` | Rust version (e.g., `1.83.0`) | Latest stable |
| `NODE_VERSION` | Node.js version (e.g., `22.11.0`) | Latest from nodejs.org |

## Usage Examples

### Basic Usage

```bash
# Build with latest versions
docker build -t shai-mega .

# Run interactive shell
docker run -it --rm shai-mega

# Run with volume mount
docker run -it --rm -v $(pwd):/src shai-mega
```

### Build with Specific Versions

```bash
# Build with Go 1.24.0 and latest Rust/Node
docker build --build-arg GO_VERSION=1.24.0 -t shai-mega:go1.24 .

# Build with specific versions for all languages
docker build \
  --build-arg GO_VERSION=1.24.0 \
  --build-arg RUST_VERSION=1.83.0 \
  --build-arg NODE_VERSION=20.11.0 \
  -t shai-mega:pinned .
```

### Development Workflow

```bash
# Start container with project mounted
docker run -it --rm \
  -v $(pwd):/src \
  -w /src \
  shai-mega

# Inside container
go build ./...
cargo build
npm install
python3 -m venv venv && source venv/bin/activate
```

## Layer Structure

The Dockerfile is organized into distinct layers for optimal caching:

1. **Base system packages** - apt packages and build tools
2. **Go installation** - Go toolchain
3. **Rust installation** - Rust compiler and cargo
4. **Node.js installation** - Node runtime and npm
5. **Python verification** - Verify Python installation
6. **Java verification** - Verify JDK installation
7. **npm package managers** - npm, yarn, pnpm
8. **AI CLI tools** - AI assistant CLIs
9. **Playwright** - Browser automation with Chromium
10. **System configuration** - Locale generation
11. **Profile.d scripts** - Environment variable configuration

This structure ensures that:
- Changing npm package versions doesn't rebuild Go/Rust/Node layers
- Changing Go version doesn't rebuild Rust/Node layers
- Language installations are independent and cached separately
- Configuration is managed via standard Linux patterns

## Environment Variables

All environment variables are configured using Linux standard patterns via `/etc/profile.d` scripts and `/etc/zsh/zshenv` for zsh compatibility.

### Configuration Files

#### `/etc/profile.d/10-locale.sh`
- `LANG=en_US.UTF-8`
- `LC_ALL=en_US.UTF-8`
- `LANGUAGE=en_US:en`

#### `/etc/profile.d/10-proxy.sh`
**Note:** Proxy settings only apply to non-root users (uid != 0)
- `HTTP_PROXY=http://127.0.0.1:8888`
- `HTTPS_PROXY=http://127.0.0.1:8888`
- `NO_PROXY=localhost,127.0.0.1,::1`
- `http_proxy=http://127.0.0.1:8888`
- `https_proxy=http://127.0.0.1:8888`
- `no_proxy=localhost,127.0.0.1,::1`

#### `/etc/profile.d/10-lang-paths.sh`
- `RUSTUP_HOME=/usr/local/rustup`
- `CARGO_HOME=/usr/local/cargo`
- `PATH=/usr/local/go/bin:/usr/local/cargo/bin:$PATH`
- `PLAYWRIGHT_BROWSERS_PATH=/ms-playwright`

## Supervisor

The image includes supervisor for process management. The `/etc/supervisor/conf.d/` directory is available for adding service configurations.

Example usage:
```bash
# Create a supervisor config for your service
cat > /etc/supervisor/conf.d/myservice.conf <<EOF
[program:myservice]
command=/path/to/your/service
autostart=true
autorestart=true
stdout_logfile=/var/log/supervisor/myservice.log
stderr_logfile=/var/log/supervisor/myservice.err
EOF

# Start supervisor
supervisord -n
```

## Build Performance

### Multi-stage Build

The build uses a separate `version-fetcher` stage that:
1. Queries upstream sources for latest versions
2. Stores versions in files
3. Copies version files into main build

This approach ensures:
- **Efficient caching**: Version files only change when upstream versions change
- **Automatic updates**: Rebuilding fetches the latest versions
- **Reproducibility**: Build args allow pinning for CI/CD

### Cache Optimization Tips

```bash
# Use BuildKit for better caching
DOCKER_BUILDKIT=1 docker build .

# Use cache from previous builds
docker build --cache-from shai-mega:latest .

# Build with multiple cache sources
docker build \
  --cache-from shai-mega:latest \
  --cache-from shai-mega:dev \
  .
```

## License

This Dockerfile and configuration are provided as-is for development purposes.
