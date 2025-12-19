# shai-base: Minimal Debian image with shai runtime dependencies
# This image provides the core system packages required for shai sandboxing
# Image: colony-2/shai-base

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="colony-2/shai-base" \
      org.opencontainers.image.description="Minimal Debian base with shai runtime dependencies (supervisor, dnsmasq, tinyproxy, iptables)" \
      org.opencontainers.image.title="shai-base"

ARG DEBIAN_FRONTEND=noninteractive

# Ensure predictable shell behavior for RUN chains
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

## Install shai runtime dependencies
# Core packages required for shai bootstrap and sandboxing:
# - bash: Shell for bootstrap script
# - ca-certificates: SSL/TLS certificates for HTTPS
# - coreutils: Basic commands (mkdir, chmod, chown, rm, cp, install, etc.)
# - curl: HTTP client for downloads
# - dnsmasq: DNS server for domain filtering
# - iptables: Firewall for network egress control
# - iproute2: Network utilities (ss, ip)
# - iputils-ping: Network diagnostics (ping)
# - jq: JSON processor
# - net-tools: Legacy network utilities (netstat)
# - passwd: User management utilities (useradd, usermod, etc.)
# - procps: Process utilities (ps, top, etc.)
# - sed: Stream editor for text processing
# - supervisor: Process supervisor for background services
# - tinyproxy: HTTP/HTTPS proxy for allow-listed traffic
# - util-linux: System utilities (runuser, su, etc.)
RUN set -euxo pipefail \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
       bash \
       ca-certificates \
       coreutils \
       curl \
       dnsmasq \
       iptables \
       iproute2 \
       iputils-ping \
       jq \
       net-tools \
       passwd \
       procps \
       sed \
       supervisor \
       tinyproxy \
       util-linux \
    && rm -rf /var/lib/apt/lists/*

# Create standard directories
RUN mkdir -p /src

WORKDIR /src
