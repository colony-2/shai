# shai-mega: Debian slim with dev tools and CLIs
# Includes: Go, Rust (stable), Python, Node, C/C++, Java, jq, git
# Default working directory: /src
# Image: colony-2/shai-mega

## Build args for base image selection
ARG BASE_IMAGE=debian:bookworm-slim

## Build stage to determine latest versions
FROM ${BASE_IMAGE} AS version-fetcher

ARG DEBIAN_FRONTEND=noninteractive
ARG CACHE_BUST

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates jq && \
    rm -rf /var/lib/apt/lists/*

# Cache-busting layer to force version refresh when needed
RUN echo "Cache bust: ${CACHE_BUST:-default}"

# Fetch latest versions of languages
RUN curl -s https://go.dev/dl/?mode=json | jq -r '.[0].version' | sed 's/^go//' > /tmp/go.version && \
    curl -s https://nodejs.org/dist/index.json | jq -r '.[0].version' | sed 's/^v//' > /tmp/node.version && \
    curl -s https://static.rust-lang.org/dist/channel-rust-stable.toml | grep -A 5 '^\[pkg\.rust\]' | grep '^version = ' | head -1 | cut -d'"' -f2 | awk '{print $1}' > /tmp/rust.version

# Fetch latest versions of npm packages
RUN curl -s https://registry.npmjs.org/npm | jq -r '.["dist-tags"].latest' > /tmp/npm.version && \
    curl -s https://registry.npmjs.org/yarn | jq -r '.["dist-tags"].latest' > /tmp/yarn.version && \
    curl -s https://registry.npmjs.org/pnpm | jq -r '.["dist-tags"].latest' > /tmp/pnpm.version && \
    curl -s https://registry.npmjs.org/@openai/codex | jq -r '.["dist-tags"].latest' > /tmp/codex.version && \
    curl -s https://registry.npmjs.org/@google/gemini-cli | jq -r '.["dist-tags"].latest' > /tmp/gemini.version && \
    curl -s https://registry.npmjs.org/@anthropic-ai/claude-code | jq -r '.["dist-tags"].latest' > /tmp/claude.version && \
    curl -s https://registry.npmjs.org/@moonrepo/cli | jq -r '.["dist-tags"].latest' > /tmp/moon.version && \
    curl -s https://registry.npmjs.org/playwright | jq -r '.["dist-tags"].latest' > /tmp/playwright.version

## Main build stage
ARG BASE_IMAGE=debian:bookworm-slim
FROM ${BASE_IMAGE}

LABEL org.opencontainers.image.source="colony-2/shai-mega" \
      org.opencontainers.image.description="Slim Debian dev image with Go, Rust, Python, Node, C/C++, Java, jq, git, and common AI CLIs" \
      org.opencontainers.image.title="shai-mega"

## Versions - no defaults, will use latest unless overridden
ARG DEBIAN_FRONTEND=noninteractive
ARG GO_VERSION
ARG RUST_VERSION
ARG NODE_VERSION

# Ensure predictable shell behavior for RUN chains
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH="/usr/local/cargo/bin:/usr/local/go/bin:${PATH}"

## Layer 1: Base system packages and build tools
RUN set -euxo pipefail \
    && apt-get update \
    && icu_pkg=$(apt-cache search -n '^libicu[0-9][0-9]$' | awk '{print $1}' | sort -V | tail -n1) \
    && lttng_pkg=$(apt-cache search -n '^liblttng-ust[0-9].*$' | awk '{print $1}' | sort -V | tail -n1) \
    && apt-get install -y --no-install-recommends \
       apt-utils ca-certificates curl wget gnupg gnupg2 dirmngr unzip xz-utils tar \
       git jq bash-completion iproute2 procps lsof htop net-tools psmisc tree rsync \
       bzip2 zip nano vim-tiny less lsb-release apt-transport-https dialog \
       libc6 libgcc1 libkrb5-3 libgssapi-krb5-2 libstdc++6 zlib1g locales sudo \
       ncdu man-db strace manpages manpages-dev init-system-helpers libssl3 zsh iptables supervisor inotify-tools \
       build-essential pkg-config clang libclang-dev \
       python3 python3-pip python3-venv python3-dev \
       default-jdk \
       openssh-client \
       sshpass \
       coreutils \
       tinyproxy \
       dnsmasq \
       bash \
       iputils-ping \
       passwd \
       sed \
       util-linux \
       "$icu_pkg" "$lttng_pkg" \
    && rm -rf /var/lib/apt/lists/*

## Layer 2: Go installation
COPY --from=version-fetcher /tmp/go.version /tmp/go-latest.version

RUN set -euxo pipefail \
    && GO_VER=${GO_VERSION:-$(cat /tmp/go-latest.version)} \
    && echo "Installing Go ${GO_VER}" \
    && arch=$(dpkg --print-architecture) \
    && case "$arch" in amd64) GOARCH=amd64 ;; arm64) GOARCH=arm64 ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac \
    && curl -fsSL "https://go.dev/dl/go${GO_VER}.linux-${GOARCH}.tar.gz" -o /tmp/go.tgz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm -f /tmp/go.tgz \
    && ln -sf /usr/local/go/bin/go /usr/local/bin/go \
    && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt \
    && go version

## Layer 3: Rust installation
COPY --from=version-fetcher /tmp/rust.version /tmp/rust-latest.version

RUN set -euxo pipefail \
    && RUST_VER=${RUST_VERSION:-$(cat /tmp/rust-latest.version)} \
    && echo "Installing Rust ${RUST_VER}" \
    && export RUSTUP_HOME=/usr/local/rustup CARGO_HOME=/usr/local/cargo \
    && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs -o /tmp/rustup.sh \
    && sh /tmp/rustup.sh -y --default-toolchain "${RUST_VER}" --profile minimal --no-modify-path \
    && rm -f /tmp/rustup.sh \
    && chmod -R a+rX /usr/local/rustup /usr/local/cargo \
    && mkdir -p /usr/local/cargo/registry \
    && chmod 0777 /usr/local/cargo/registry \
    && /usr/local/cargo/bin/rustc --version && /usr/local/cargo/bin/cargo --version

## Layer 4: Node.js installation
COPY --from=version-fetcher /tmp/node.version /tmp/node-latest.version

RUN set -euxo pipefail \
    && NODE_VER=${NODE_VERSION:-$(cat /tmp/node-latest.version)} \
    && echo "Installing Node.js ${NODE_VER}" \
    && arch=$(dpkg --print-architecture) \
    && case "$arch" in amd64) NODEARCH=x64 ;; arm64) NODEARCH=arm64 ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac \
    && curl -fsSL "https://nodejs.org/dist/v${NODE_VER}/node-v${NODE_VER}-linux-${NODEARCH}.tar.xz" -o /tmp/node.tar.xz \
    && tar -xJf /tmp/node.tar.xz -C /usr/local --strip-components=1 \
    && rm -f /tmp/node.tar.xz \
    && node --version && npm --version

## Layer 5: Python verification
RUN python3 --version && python3 -m pip --version

## Layer 6: Java verification
RUN java -version || true

## Layer 7: Core npm package managers with versions
COPY --from=version-fetcher /tmp/npm.version /tmp/yarn.version /tmp/pnpm.version /tmp/

RUN set -euxo pipefail \
    && NPM_VER=$(cat /tmp/npm.version) \
    && YARN_VER=$(cat /tmp/yarn.version) \
    && PNPM_VER=$(cat /tmp/pnpm.version) \
    && echo "Installing npm@${NPM_VER} yarn@${YARN_VER} pnpm@${PNPM_VER}" \
    && npm -g install "npm@${NPM_VER}" "yarn@${YARN_VER}" "pnpm@${PNPM_VER}" \
    && npm cache clean --force \
    && npm --version && yarn --version && pnpm --version

## Layer 8: AI CLI tools with versions
COPY --from=version-fetcher /tmp/codex.version /tmp/gemini.version /tmp/claude.version /tmp/moon.version /tmp/

RUN set -euxo pipefail \
    && CODEX_VER=$(cat /tmp/codex.version) \
    && GEMINI_VER=$(cat /tmp/gemini.version) \
    && CLAUDE_VER=$(cat /tmp/claude.version) \
    && MOON_VER=$(cat /tmp/moon.version) \
    && echo "Installing AI CLIs: codex@${CODEX_VER} gemini@${GEMINI_VER} claude@${CLAUDE_VER} moon@${MOON_VER}" \
    && npm -g install "@openai/codex@${CODEX_VER}" "@google/gemini-cli@${GEMINI_VER}" "@anthropic-ai/claude-code@${CLAUDE_VER}" "@moonrepo/cli@${MOON_VER}" \
    && npm cache clean --force \
    && codex -h >/dev/null 2>&1 || true \
    && gemini -h >/dev/null 2>&1 || true \
    && claude -h >/dev/null 2>&1 || claude-code -h >/dev/null 2>&1 || true

## Layer 9: Playwright installation with dependencies
COPY --from=version-fetcher /tmp/playwright.version /tmp/playwright.version

RUN set -euxo pipefail \
    && PLAYWRIGHT_VER=$(cat /tmp/playwright.version) \
    && echo "Installing Playwright ${PLAYWRIGHT_VER}" \
    && mkdir -p /ms-playwright \
    && chmod -R a+rx /ms-playwright \
    && npm -g install "playwright@${PLAYWRIGHT_VER}" \
    && npx playwright install --with-deps chromium \
    && chmod -R a+rx /ms-playwright \
    && npm cache clean --force

## Layer 10: System configuration (locale, environment)
RUN set -euxo pipefail \
    && if [ -f /etc/locale.gen ]; then sed -i 's/^# \?en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen && locale-gen; fi \
    && mkdir -p /src

## Layer 11: Profile.d environment scripts
COPY 10-locale.sh /etc/profile.d/10-locale.sh
COPY 10-lang-paths.sh /etc/profile.d/10-lang-paths.sh

RUN set -euxo pipefail \
    && chmod 0644 /etc/profile.d/10-locale.sh \
    && chmod 0644 /etc/profile.d/10-lang-paths.sh \
    && if [ -f /etc/zsh/zshenv ]; then \
        cat /etc/profile.d/10-locale.sh >> /etc/zsh/zshenv; \
        cat /etc/profile.d/10-lang-paths.sh >> /etc/zsh/zshenv; \
    fi

WORKDIR /src
