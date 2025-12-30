---
title: Apply Rules
weight: 3
---

**Apply Rules** map workspace paths to [resource sets](../resource-sets), automatically activating the right resources based on where agents are working.

## What Are Apply Rules?

Apply rules answer the question: *"When an agent works in this directory, which resource sets should be available?"*

They're defined in `.shai/config.yaml`:

```yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: frontend
    resources: [npm-registries, playwright]

  - path: backend/payments
    resources: [stripe-api, database-access]
```

## Why Apply Rules?

Without apply rules, you'd need to manually specify resource sets every time:

```bash
# Without apply rules (tedious!)
shai -rw frontend/components --resource-set npm-registries --resource-set playwright
```

With apply rules, Shai automatically resolves the right resources:

```bash
# With apply rules (automatic!)
shai -rw frontend/components
# Automatically gets: base-allowlist, npm-registries, playwright
```

## How Rules Work

### Path Matching

Rules match **workspace-relative paths**:

```yaml
apply:
  - path: ./              # Matches everything
    resources: [base]

  - path: frontend        # Matches frontend/**/*
    resources: [npm]

  - path: backend/auth    # Matches backend/auth/**/*
    resources: [database]
```

When you run:
```bash
shai -rw backend/auth/handlers
```

Shai finds all rules that match `backend/auth/handlers`:
1. `./` ✅ (matches everything)
2. `frontend` ❌ (doesn't match)
3. `backend/auth` ✅ (matches)

Result: `[base, database]`

### Aggregation

Resource sets from **all matching rules** are combined:

```yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: backend
    resources: [database-access]

  - path: backend/payments
    resources: [stripe-api, payment-webhooks]
```

Running `shai -rw backend/payments`:
- `./` matches → `[base-allowlist]`
- `backend` matches → `[database-access]`
- `backend/payments` matches → `[stripe-api, payment-webhooks]`

**Final result:** `[base-allowlist, database-access, stripe-api, payment-webhooks]`

{{< callout type="info" >}}
Resource sets are deduplicated. If multiple rules specify the same set, it's only included once.
{{< /callout >}}

### Evaluation Order

Rules are evaluated **top to bottom**, but all matching rules are applied:

```yaml
apply:
  - path: ./
    resources: [base]

  - path: backend
    resources: [api-tools]

  - path: backend        # Duplicate path is fine
    resources: [testing]
```

All matching `backend` rules are applied: `[base, api-tools, testing]`

## Image Overrides

Apply rules can also override the container image:

```yaml
apply:
  - path: ./
    resources: [base-allowlist]
    # Uses default image from top-level config

  - path: infrastructure
    resources: [deployment-tools]
    image: ghcr.io/my-org/devops:latest

  - path: ml-training
    resources: [gpu-access]
    image: ghcr.io/my-org/pytorch-gpu:latest
```

When you run `shai -rw infrastructure/terraform`, it uses the `devops:latest` image instead of the default.

### Image Override Rules

1. **More specific paths take precedence:**
   ```yaml
   apply:
     - path: ./
       image: ghcr.io/colony-2/shai-mega

     - path: backend/payments
       image: ghcr.io/my-org/payments-dev:latest
   ```
   Running `shai -rw backend/payments` uses `payments-dev:latest`.

2. **Root path (`.` or `./`) cannot override the image:**
   ```yaml
   apply:
     - path: ./
       image: foo:latest    # ❌ Ignored! Use top-level `image` key instead
   ```

3. **CLI flag takes ultimate precedence:**
   ```bash
   shai -rw infrastructure --image custom:latest
   # Uses custom:latest regardless of config
   ```

## Path Syntax

### Relative Paths

All paths in apply rules are relative to the workspace root:

```yaml
apply:
  - path: ./            # Root (matches everything)
  - path: .             # Same as ./
  - path: src           # Matches src/**/*
  - path: src/backend   # Matches src/backend/**/*
```

### Subdirectory Matching

Paths match themselves and all subdirectories:

```yaml
apply:
  - path: backend
    resources: [api-tools]
```

This matches:
- `backend/` ✅
- `backend/auth/` ✅
- `backend/payments/handlers/` ✅
- `frontend/` ❌

### Exact vs Prefix Matching

Currently, Shai uses **prefix matching**. There's no way to match a path *exactly* without matching its subdirectories.

To work around this, use more specific paths:

```yaml
apply:
  - path: backend
    resources: [common-backend-tools]

  - path: backend/payments
    resources: [payment-specific-tools]
```

## Multiple Target Paths

When you specify multiple `-rw` paths, Shai resolves resources for **each path** and aggregates:

```bash
shai -rw frontend/app -rw backend/api
```

```yaml
apply:
  - path: ./
    resources: [base]

  - path: frontend
    resources: [npm]

  - path: backend
    resources: [database]
```

**Resolution:**
1. `frontend/app` matches: `[base, npm]`
2. `backend/api` matches: `[base, database]`
3. Aggregate (deduplicate): `[base, npm, database]`

## Common Patterns

### Layered Resources

Build up resources from general to specific:

```yaml
apply:
  # Everything gets base allowlist
  - path: ./
    resources: [base-allowlist]

  # All frontend code gets npm + playwright
  - path: frontend
    resources: [npm-registries, playwright]

  # Specific frontend app gets additional CDN access
  - path: frontend/marketing-site
    resources: [cdn-access]
```

### Per-Service Resources

In a microservices repo:

```yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: services/auth
    resources: [database-access, jwt-tools]

  - path: services/payments
    resources: [database-access, stripe-api, payment-testing]

  - path: services/notifications
    resources: [smtp-access, push-notification-api]
```

### Per-Environment Resources

Different paths for different environments:

```yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: infrastructure/staging
    resources: [staging-credentials, staging-k8s]

  - path: infrastructure/production
    resources: [production-credentials, production-k8s]
```

## Debugging Apply Rules

Use `--verbose` to see which rules match:

```bash
shai -rw backend/payments --verbose
```

Output shows:
- Matched rules
- Resolved resource sets
- Final aggregated resources
- Which image is selected

## Best Practices

### ✅ Do

- Start with a root `./` rule for common resources
- Layer more specific rules on top
- Use descriptive path names that match your project structure
- Keep rules simple and predictable
- Document complex rule hierarchies

### ❌ Don't

- Create conflicting rules that make resolution unclear
- Over-specify rules for every single directory
- Use image overrides on the root path
- Assume rules are evaluated in a specific order (always think aggregation)

## Examples

### Monorepo with Multiple Teams

```yaml
apply:
  # Base for everyone
  - path: ./
    resources: [base-allowlist, git-ssh]

  # Web team
  - path: packages/web-app
    resources: [npm-registries, playwright]

  - path: packages/mobile-app
    resources: [npm-registries, react-native-tools]

  # Backend team
  - path: services/api
    resources: [database-access, api-testing]

  - path: services/workers
    resources: [redis-access, queue-monitoring]

  # DevOps team
  - path: infrastructure
    resources: [cloud-apis, k8s-access]
    image: ghcr.io/my-org/devops:latest
```

### ML Project

```yaml
apply:
  # Base
  - path: ./
    resources: [base-allowlist]

  # Data preparation
  - path: data
    resources: [s3-access, data-processing]

  # Model training
  - path: models
    resources: [gpu-access, wandb-api, model-registry]
    image: ghcr.io/my-org/pytorch-gpu:latest

  # Serving
  - path: serving
    resources: [model-registry, k8s-access]
```

### Full-Stack Application

```yaml
apply:
  # Shared base
  - path: ./
    resources: [base-allowlist]

  # Frontend
  - path: frontend
    resources: [npm-registries, cdn-access]

  # Backend API
  - path: backend/api
    resources: [database-access, redis-access]

  # Backend workers
  - path: backend/workers
    resources: [redis-access, smtp-access]

  # Database migrations
  - path: backend/migrations
    resources: [database-admin-access]

  # Infrastructure
  - path: infrastructure
    resources: [terraform-cloud, aws-apis]
```

## Next Steps

- Learn about [Resource Sets](../resource-sets) to define collections of resources
- Understand [Cellular Development](../cellular-development) to make the most of apply rules
- See [Configuration Reference](/docs/configuration) for complete schema details
