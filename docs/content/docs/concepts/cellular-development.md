---
title: Cellular Development
weight: 1
---

Shai is built around the concept of **cellular software development** - giving agents constrained access to individual components rather than entire repositories.

## The Problem

Traditional AI agent workflows give agents access to your entire codebase:

```
my-app/
├── frontend/          ← Agent can modify
├── backend/           ← Agent can modify
├── infrastructure/    ← Agent can modify
├── docs/              ← Agent can modify
└── .env               ← Agent can modify (dangerous!)
```

This creates several issues:

- **Large blast radius**: A bug in the agent can damage unrelated code
- **Security risks**: Agents can accidentally commit credentials
- **Context overload**: Agents waste tokens understanding irrelevant code
- **Merge conflicts**: Multiple agents working in parallel conflict

## The Cellular Approach

With Shai, you give agents access to **specific components**:

```
my-app/
├── frontend/          ← Read-only
├── backend/
│   └── auth/          ← Writable (agent works here)
├── infrastructure/    ← Read-only
└── docs/              ← Read-only
```

The agent can:
- ✅ Read the entire codebase for context
- ✅ Modify only `backend/auth/`
- ❌ Change unrelated components
- ❌ Accidentally modify config files

## Benefits

### 1. Reduced Blast Radius

If an agent makes a mistake, it's contained to the specific component it's working on.

```bash
# Agent can only affect the auth module
shai -rw backend/auth -- claude-code
```

### 2. Parallel Workflows

Multiple agents can work simultaneously on different components:

```bash
# Terminal 1: Agent working on auth
shai -rw backend/auth -- claude-code

# Terminal 2: Agent working on payments
shai -rw backend/payments -- gemini-cli

# No conflicts! Each agent has its own cell.
```

### 3. Context Efficiency

Agents focus on relevant code instead of trying to understand an entire monorepo.

### 4. Security

Credentials and configuration files stay read-only unless explicitly needed.

```bash
# Agent can modify code but not secrets
shai -rw src/components
# .env files remain read-only
```

## Target Paths

When you run Shai, you specify **target paths** with the `-rw` flag:

```bash
shai -rw <path1> -rw <path2> ...
```

These paths determine:
1. Which directories the agent can modify
2. Which [resource sets](../resource-sets) and [apply rules](../apply-rules) are activated

## Examples

### Frontend Component

```bash
shai -rw src/components/LoginForm -- claude-code
```

Agent can:
- Modify `src/components/LoginForm/`
- Read the rest of the codebase
- Access frontend development resources (npm, etc.)

### Backend API Module

```bash
shai -rw internal/api/users -rw internal/api/auth
```

Agent can modify two related modules while keeping the rest read-only.

### Testing

```bash
shai -rw tests/integration
```

Agent can write tests without touching implementation code.

### Documentation

```bash
shai -rw docs
```

Agent can update docs without risking code changes.

## Monorepo Pattern

For monorepos, cellular development shines:

```
monorepo/
├── packages/
│   ├── web-app/       ← Cell 1
│   ├── mobile-app/    ← Cell 2
│   ├── shared-ui/     ← Cell 3
│   └── api-client/    ← Cell 4
└── services/
    ├── auth/          ← Cell 5
    ├── payments/      ← Cell 6
    └── notifications/ ← Cell 7
```

Each package or service is a cell. Agents work on one cell at a time:

```bash
# Working on the web app
shai -rw packages/web-app -- claude-code

# Working on the auth service
shai -rw services/auth -- codex
```

## When to Use Multiple Target Paths

Sometimes you need to make an agent work across multiple directories:

```bash
# Agent needs to modify both implementation and tests
shai -rw src/auth -rw tests/auth
```

**Guidelines:**
- Keep target paths related and cohesive
- Avoid giving access to the entire repo (`-rw .` defeats the purpose)
- Think about the agent's task scope

## Read-Only Context

Even though agents can only **write** to target paths, they can **read** the entire workspace:

```bash
shai -rw backend/auth
# Inside the sandbox:
# - Can write to /src/backend/auth
# - Can read /src/frontend, /src/backend/*, etc.
```

This allows agents to:
- Understand how components integrate
- Follow existing patterns
- Avoid breaking interfaces

## Cellular + Resource Sets

Cellular development becomes even more powerful when combined with [resource sets](../resource-sets) and [apply rules](../apply-rules).

Different components can have different resources:

```yaml
# .shai/config.yaml
apply:
  - path: ./
    resources: [base-allowlist]

  - path: frontend
    resources: [npm-registries, playwright]

  - path: backend/payments
    resources: [stripe-api, payment-testing]

  - path: infrastructure
    resources: [cloud-apis, deployment-tools]
    image: ghcr.io/my-org/devops:latest
```

When you run `shai -rw backend/payments`, you automatically get the Stripe API and payment testing resources.

## Best Practices

### ✅ Do

- Start with the smallest necessary scope
- Use cellular development even for solo projects
- Combine related directories when they must change together
- Let agents read the full codebase for context

### ❌ Don't

- Give agents root-level write access (`-rw .`)
- Make unrelated directories writable together
- Assume agents need write access everywhere
- Forget that read access provides context

## Next Steps

- Learn about [Resource Sets](../resource-sets) to control what agents can access
- Understand [Apply Rules](../apply-rules) to map paths to resources
- See [Examples](/docs/examples) of cellular development in action
