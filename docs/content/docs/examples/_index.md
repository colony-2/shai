---
title: Examples
weight: 5
---

Real-world configuration examples for common use cases.

## Quick Examples

### Frontend Development

```yaml
resources:
  frontend:
    http:
      - cdn.jsdelivr.net
      - unpkg.com
      - fonts.googleapis.com
    mounts:
      - source: ${{ env.HOME }}/.npm
        target: /home/${{ conf.TARGET_USER }}/.npm
        mode: rw

apply:
  - path: ./
    resources: [base-allowlist]
  - path: frontend
    resources: [frontend]
```

```bash
shai -rw frontend/src -- claude
```

### Backend API

```yaml
resources:
  api-dev:
    vars:
      - source: DATABASE_URL
    mounts:
      - source: ${{ env.HOME }}/.cargo
        target: /home/${{ conf.TARGET_USER }}/.cargo
        mode: rw
    ports:
      - host: localhost
        port: 5432

apply:
  - path: backend/api
    resources: [api-dev]
```

```bash
shai -rw backend/api -- cargo build
```

### ML/AI Development

```yaml
resources:
  ml-tools:
    http:
      - huggingface.co
    vars:
      - source: HUGGINGFACE_TOKEN
    mounts:
      - source: ${{ env.HOME }}/.cache/huggingface
        target: /home/${{ conf.TARGET_USER }}/.cache/huggingface
        mode: rw

apply:
  - path: ml
    resources: [ml-tools]
    image: ghcr.io/my-org/pytorch-gpu:latest
```

### Deployment

```yaml
resources:
  deploy:
    vars:
      - source: AWS_ACCESS_KEY_ID
      - source: AWS_SECRET_ACCESS_KEY
    http:
      - amazonaws.com
    calls:
      - name: deploy-to-staging
        description: Deploy to staging
        command: /usr/local/bin/deploy.sh
        allowed-args: '^--env=staging$'

apply:
  - path: infrastructure
    resources: [deploy]
```

## Agent Integration

### Claude Code

```bash
# Basic usage
shai -rw src -- claude --dangerously-bypass-approvals-and-sandbox

# With specific component
shai -rw src/components/Auth -- claude --dangerously-bypass-approvals-and-sandbox
```

### Codex

```bash
shai -rw backend/services -- codex --dangerously-bypass-approvals-and-sandbox
```

### Gemini CLI

```bash
shai -rw packages/web-app -- gemini-cli
```

## Monorepo Pattern

```yaml
apply:
  - path: ./
    resources: [base-allowlist, git-ssh]

  - path: packages/web-app
    resources: [frontend-tools]

  - path: packages/api-client
    resources: [typescript-tools]

  - path: services/auth
    resources: [backend-tools, database-access]

  - path: services/payments
    resources: [backend-tools, database-access, stripe-api]

  - path: infrastructure
    resources: [cloud-tools]
    image: ghcr.io/my-org/devops:latest
```

Usage:
```bash
# Work on web app
shai -rw packages/web-app

# Work on auth service
shai -rw services/auth

# Work on infrastructure
shai -rw infrastructure
```

## See Also

- [Configuration Reference](/docs/configuration) for complete schema
- [Core Concepts](/docs/concepts) for understanding resource sets and apply rules
- [Docker Images](/docs/docker-images) for custom image examples
