# shai-mega: Debian slim with dev tools and CLIs
# Includes: Go, Rust (stable), Python, Node, C/C++, Java, jq, git, Bazel, and cloud/Kubernetes CLIs
# Default working directory: /src
# Image: colony-2/shai-mega

## Build args for base image selection
ARG BASE_IMAGE=debian:bookworm-slim

## Build stage to determine latest versions
FROM ${BASE_IMAGE} AS version-fetcher

ARG DEBIAN_FRONTEND=noninteractive
ARG CACHE_BUST

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates gzip jq && \
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
    curl -s https://registry.npmjs.org/@colony2/c2m | jq -r '.["dist-tags"].latest' > /tmp/c2m.version && \
    curl -s https://registry.npmjs.org/@colony2/c2j | jq -r '.["dist-tags"].latest' > /tmp/c2j.version && \
    curl -s https://registry.npmjs.org/@moonrepo/cli | jq -r '.["dist-tags"].latest' > /tmp/moon.version && \
    curl -s https://registry.npmjs.org/wrangler | jq -r '.["dist-tags"].latest' > /tmp/wrangler.version && \
    curl -s https://registry.npmjs.org/playwright | jq -r '.["dist-tags"].latest' > /tmp/playwright.version

# Fetch latest versions of infra and platform CLIs
RUN curl --retry 5 --retry-all-errors --retry-delay 2 -fsSL 'https://api.github.com/repos/bazelbuild/bazel/releases?per_page=30' | \
      jq -er '[.[] | select(.draft | not) | select(.prerelease | not) | . as $release | ($release.tag_name | ltrimstr("v")) as $version | select(([$release.assets[].name] | index("bazel-\($version)-linux-x86_64")) and ([$release.assets[].name] | index("bazel-\($version)-linux-arm64"))) | $version][0] // error("no stable Bazel release with linux x86_64 and arm64 assets found")' > /tmp/bazel.version && \
    curl -fsSL https://api.github.com/repos/opentofu/opentofu/releases/latest | jq -r '.tag_name | ltrimstr("v")' > /tmp/opentofu.version && \
    curl -fsSL https://api.github.com/repos/aws/aws-cli/tags?per_page=1 | jq -r '.[0].name' > /tmp/awscli.version && \
    curl -fsSL https://dl.google.com/dl/cloudsdk/channels/rapid/components-2.json | jq -r '.version' > /tmp/gcloud.version && \
    curl -fsSL https://pypi.org/pypi/azure-cli/json | jq -r '.info.version' > /tmp/azure-cli.version && \
    curl -fsSL https://pypi.org/pypi/uv/json | jq -r '.info.version' > /tmp/uv.version && \
    curl -fsSL https://api.github.com/repos/pulumi/pulumi/releases/latest | jq -r '.tag_name | ltrimstr("v")' > /tmp/pulumi.version && \
    curl -fsSL https://api.github.com/repos/helm/helm/releases/latest | jq -r '.tag_name | ltrimstr("v")' > /tmp/helm.version && \
    curl -fsSL https://api.github.com/repos/kubernetes-sigs/kustomize/releases/latest | jq -r '.tag_name | split("/")[-1] | ltrimstr("v")' > /tmp/kustomize.version

RUN KUBECTL_MINOR=$(curl -fsSL https://api.github.com/repos/kubernetes/kubernetes/releases?per_page=20 | jq -r 'map(select(.prerelease | not)) | .[0].tag_name | ltrimstr("v")' | awk -F. '{print $1 "." $2}') && \
    curl -fsSL "https://pkgs.k8s.io/core:/stable:/v${KUBECTL_MINOR}/deb/Packages.gz" | gzip -dc | awk '$1 == "Package:" && $2 == "kubectl" {in_pkg=1; next} /^$/ {in_pkg=0} in_pkg && $1 == "Version:" {print $2}' | sort -V | tail -n1 > /tmp/kubectl-package.version && \
    sed 's/-.*//' /tmp/kubectl-package.version > /tmp/kubectl.version

## Main build stage
ARG BASE_IMAGE=debian:bookworm-slim
FROM ${BASE_IMAGE}

LABEL org.opencontainers.image.source="colony-2/shai-mega" \
      org.opencontainers.image.description="Slim Debian dev image with Go, Rust, Python, Node, C/C++, Java, jq, git, Bazel, and common AI, cloud, IaC, and Kubernetes CLIs" \
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
    PATH="/usr/local/cargo/bin:/usr/local/go/bin:${PATH}" \
    CLOUDSDK_PYTHON=/usr/bin/python3

## Layer 1: Base system packages and build tools
RUN set -euxo pipefail \
    && apt-get update \
    && icu_pkg=$(apt-cache search -n '^libicu[0-9][0-9]$' | awk '{print $1}' | sort -V | tail -n1) \
    && lttng_pkg=$(apt-cache search -n '^liblttng-ust[0-9].*$' | awk '{print $1}' | sort -V | tail -n1) \
    && apt-get install -y --no-install-recommends \
       apt-utils ca-certificates curl wget gnupg gnupg2 dirmngr unzip xz-utils tar \
       git jq bash-completion iproute2 procps lsof htop net-tools psmisc tree rsync \
       bzip2 zip nano vim-tiny less lsb-release apt-transport-https dialog \
       ripgrep \
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

## Layer 6: uv installation with version pinning
COPY --from=version-fetcher /tmp/uv.version /tmp/uv.version

RUN set -euxo pipefail \
    && UV_VER=$(cat /tmp/uv.version) \
    && echo "Installing uv ${UV_VER}" \
    && rm -rf /opt/uv \
    && python3 -m venv /opt/uv \
    && /opt/uv/bin/pip install --no-cache-dir --upgrade pip \
    && /opt/uv/bin/pip install --no-cache-dir "uv==${UV_VER}" \
    && ln -sf /opt/uv/bin/uv /usr/local/bin/uv \
    && ln -sf /opt/uv/bin/uvx /usr/local/bin/uvx \
    && command -v uv \
    && command -v uvx \
    && uv --version \
    && uvx --help >/dev/null 2>&1

## Layer 7: Java verification
RUN java -version || true

## Layer 8: Core npm package managers with versions
COPY --from=version-fetcher /tmp/npm.version /tmp/yarn.version /tmp/pnpm.version /tmp/

RUN set -euxo pipefail \
    && NPM_VER=$(cat /tmp/npm.version) \
    && YARN_VER=$(cat /tmp/yarn.version) \
    && PNPM_VER=$(cat /tmp/pnpm.version) \
    && echo "Installing npm@${NPM_VER} yarn@${YARN_VER} pnpm@${PNPM_VER}" \
    && npm -g install "npm@${NPM_VER}" "yarn@${YARN_VER}" "pnpm@${PNPM_VER}" \
    && npm cache clean --force \
    && npm --version && yarn --version && pnpm --version

## Layer 9: AI CLI tools with versions
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

## Layer 10: C2M and C2J CLIs with version pinning
COPY --from=version-fetcher /tmp/c2m.version /tmp/c2j.version /tmp/

RUN set -euxo pipefail \
    && C2M_VER=$(cat /tmp/c2m.version) \
    && C2J_VER=$(cat /tmp/c2j.version) \
    && echo "Installing @colony2/c2m ${C2M_VER} and @colony2/c2j ${C2J_VER}" \
    && npm -g install "@colony2/c2m@${C2M_VER}" "@colony2/c2j@${C2J_VER}" \
    && npm cache clean --force \
    && command -v c2m \
    && command -v c2j

## Layer 11: Wrangler CLI with version pinning
COPY --from=version-fetcher /tmp/wrangler.version /tmp/wrangler.version

RUN set -euxo pipefail \
    && WRANGLER_VER=$(cat /tmp/wrangler.version) \
    && echo "Installing wrangler ${WRANGLER_VER}" \
    && npm -g install "wrangler@${WRANGLER_VER}" \
    && npm cache clean --force \
    && wrangler --version

## Layer 12: Bazel installation with version pinning
COPY --from=version-fetcher /tmp/bazel.version /tmp/bazel.version

RUN set -euxo pipefail \
    && BAZEL_VER=$(cat /tmp/bazel.version) \
    && echo "Installing Bazel ${BAZEL_VER}" \
    && arch=$(dpkg --print-architecture) \
    && case "$arch" in amd64) BAZEL_ARCH=x86_64 ;; arm64) BAZEL_ARCH=arm64 ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac \
    && curl --retry 5 --retry-all-errors --retry-delay 2 -fsSL "https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VER}/bazel-${BAZEL_VER}-linux-${BAZEL_ARCH}" -o /usr/local/bin/bazel \
    && chmod 0755 /usr/local/bin/bazel \
    && bazel --version

## Layer 13: OpenTofu installation with version pinning
COPY --from=version-fetcher /tmp/opentofu.version /tmp/opentofu.version

RUN set -euxo pipefail \
    && OPENTOFU_VER=$(cat /tmp/opentofu.version) \
    && echo "Installing OpenTofu ${OPENTOFU_VER}" \
    && arch=$(dpkg --print-architecture) \
    && case "$arch" in amd64) OPENTOFU_ARCH=amd64 ;; arm64) OPENTOFU_ARCH=arm64 ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac \
    && curl -fsSL "https://github.com/opentofu/opentofu/releases/download/v${OPENTOFU_VER}/tofu_${OPENTOFU_VER}_${OPENTOFU_ARCH}.deb" -o /tmp/tofu.deb \
    && dpkg -i /tmp/tofu.deb \
    && rm -f /tmp/tofu.deb \
    && tofu version

## Layer 14: AWS CLI installation with version pinning
COPY --from=version-fetcher /tmp/awscli.version /tmp/awscli.version

RUN set -euxo pipefail \
    && AWSCLI_VER=$(cat /tmp/awscli.version) \
    && echo "Installing AWS CLI ${AWSCLI_VER}" \
    && rm -rf /opt/aws-cli \
    && python3 -m venv /opt/aws-cli \
    && /opt/aws-cli/bin/pip install --no-cache-dir --upgrade pip \
    && /opt/aws-cli/bin/pip install --no-cache-dir "https://github.com/aws/aws-cli/archive/refs/tags/${AWSCLI_VER}.tar.gz" \
    && ln -sf /opt/aws-cli/bin/aws /usr/local/bin/aws \
    && aws --version

## Layer 15: Google Cloud CLI installation with version pinning
COPY --from=version-fetcher /tmp/gcloud.version /tmp/gcloud.version

RUN set -euxo pipefail \
    && GCLOUD_VER=$(cat /tmp/gcloud.version) \
    && echo "Installing Google Cloud CLI ${GCLOUD_VER}" \
    && arch=$(dpkg --print-architecture) \
    && case "$arch" in amd64) GCLOUD_ARCH=x86_64 ;; arm64) GCLOUD_ARCH=arm ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac \
    && curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VER}-linux-${GCLOUD_ARCH}.tar.gz" -o /tmp/gcloud.tgz \
    && rm -rf /opt/google-cloud-sdk \
    && mkdir -p /opt \
    && tar -xzf /tmp/gcloud.tgz -C /opt \
    && rm -f /tmp/gcloud.tgz \
    && ln -sf /opt/google-cloud-sdk/bin/gcloud /usr/local/bin/gcloud \
    && ln -sf /opt/google-cloud-sdk/bin/gsutil /usr/local/bin/gsutil \
    && ln -sf /opt/google-cloud-sdk/bin/bq /usr/local/bin/bq \
    && gcloud version | head -n 1

## Layer 16: Azure CLI installation with version pinning
COPY --from=version-fetcher /tmp/azure-cli.version /tmp/azure-cli.version

RUN set -euxo pipefail \
    && AZURE_CLI_VER=$(cat /tmp/azure-cli.version) \
    && echo "Installing Azure CLI ${AZURE_CLI_VER}" \
    && rm -rf /opt/azure-cli \
    && python3 -m venv /opt/azure-cli \
    && /opt/azure-cli/bin/pip install --no-cache-dir --upgrade pip \
    && /opt/azure-cli/bin/pip install --no-cache-dir "azure-cli==${AZURE_CLI_VER}" \
    && ln -sf /opt/azure-cli/bin/az /usr/local/bin/az \
    && az version --query '"azure-cli"' -o tsv

## Layer 17: Pulumi CLI installation with version pinning
COPY --from=version-fetcher /tmp/pulumi.version /tmp/pulumi.version

RUN set -euxo pipefail \
    && PULUMI_VER=$(cat /tmp/pulumi.version) \
    && echo "Installing Pulumi ${PULUMI_VER}" \
    && arch=$(dpkg --print-architecture) \
    && case "$arch" in amd64) PULUMI_ARCH=x64 ;; arm64) PULUMI_ARCH=arm64 ;; *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; esac \
    && curl -fsSL "https://github.com/pulumi/pulumi/releases/download/v${PULUMI_VER}/pulumi-v${PULUMI_VER}-linux-${PULUMI_ARCH}.tar.gz" -o /tmp/pulumi.tgz \
    && rm -rf /opt/pulumi \
    && mkdir -p /opt \
    && tar -xzf /tmp/pulumi.tgz -C /opt \
    && rm -f /tmp/pulumi.tgz \
    && ln -sf /opt/pulumi/pulumi /usr/local/bin/pulumi \
    && pulumi version

## Layer 18: Helm installation with version pinning
COPY --from=version-fetcher /tmp/helm.version /tmp/helm.version

RUN set -euxo pipefail \
    && HELM_VER=$(cat /tmp/helm.version) \
    && echo "Installing Helm ${HELM_VER}" \
    && GOPROXY=https://proxy.golang.org,direct GOBIN=/usr/local/bin go install "helm.sh/helm/v4/cmd/helm@v${HELM_VER}" \
    && helm version --short

## Layer 19: kubectl installation with version pinning
COPY --from=version-fetcher /tmp/kubectl.version /tmp/kubectl-package.version /tmp/

RUN set -euxo pipefail \
    && KUBECTL_VER=$(cat /tmp/kubectl.version) \
    && KUBECTL_PKG_VER=$(cat /tmp/kubectl-package.version) \
    && KUBECTL_MINOR=$(printf '%s' "${KUBECTL_VER}" | awk -F. '{print $1 "." $2}') \
    && echo "Installing kubectl ${KUBECTL_VER} (${KUBECTL_PKG_VER})" \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL "https://pkgs.k8s.io/core:/stable:/v${KUBECTL_MINOR}/deb/Release.key" | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v${KUBECTL_MINOR}/deb/ /" > /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends "kubectl=${KUBECTL_PKG_VER}" \
    && rm -f /etc/apt/sources.list.d/kubernetes.list /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
    && rm -rf /var/lib/apt/lists/* \
    && kubectl version --client=true

## Layer 20: Kustomize installation with version pinning
COPY --from=version-fetcher /tmp/kustomize.version /tmp/kustomize.version

RUN set -euxo pipefail \
    && KUSTOMIZE_VER=$(cat /tmp/kustomize.version) \
    && echo "Installing Kustomize ${KUSTOMIZE_VER}" \
    && GOPROXY=https://proxy.golang.org,direct GOBIN=/usr/local/bin go install "sigs.k8s.io/kustomize/kustomize/v5@v${KUSTOMIZE_VER}" \
    && kustomize version

## Layer 21: Playwright installation with dependencies
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

## Layer 22: System configuration (locale, environment)
RUN set -euxo pipefail \
    && if [ -f /etc/locale.gen ]; then sed -i 's/^# \?en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen && locale-gen; fi \
    && mkdir -p /src

## Layer 23: Profile.d environment scripts
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
