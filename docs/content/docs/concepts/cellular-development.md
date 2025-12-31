---
title: Cellular Development
weight: 1
---

Shai is built around the concept of **cellular software development** - establishing clear boundaries and guardrails that define what agents can and cannot modify within your codebase.

## The Problem

Traditional AI agent workflows give agents write access to your entire codebase:

```
my-app/
├── frontend/          ← Agent can modify
├── backend/           ← Agent can modify
├── infrastructure/    ← Agent can modify
├── docs/              ← Agent can modify
└── .env               ← Agent can modify (dangerous!)
```

This creates several issues:

- **Scope creep**: Agents make changes beyond their assigned task ("I'll just remove that failing test in the other module")
- **Overreach**: Overeager agents push to production or modify critical infrastructure
- **No validation boundaries**: No checkpoints to verify changes at component boundaries
- **Security risks**: Agents can accidentally commit credentials or modify sensitive configs
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
- ✅ Read all workspace code for context (all files in `/src` are **visible**)
- ✅ Modify only `backend/auth/` (write access is **restricted**)
- ❌ Change unrelated components
- ❌ Accidentally modify config files
- ❌ Access credentials or environment variables (unless explicitly exposed via resource sets)

{{< callout type="info" >}}
**Cellular development is not about hiding code.** Agents can read all workspace files for context. It's about establishing **guardrails** - controlling what agents can modify and creating **validation boundaries** where changes can be reviewed. Credentials and environment variables remain protected unless explicitly exposed.
{{< /callout >}}

## Benefits

### 1. Agent Guardrails

Define clear boundaries for what each agent is allowed to modify. This prevents scope creep and overreach.

```bash
# Agent works on auth, cannot "helpfully" modify other modules
shai -rw backend/auth -- claude-code
```

**Prevents scenarios like:**
- "That test in the payments module is failing, I'll just remove it"
- "I found AWS credentials in the config, let me deploy this to production"
- "The frontend needs updating too, I'll fix that while I'm at it"

### 2. Validation Boundaries

Each cell boundary is a checkpoint where you can validate changes from the cell's perspective. Before accepting modifications to a cell, you can ensure:
- Critical functionality remains intact
- Interfaces aren't unexpectedly modified
- Security-sensitive code hasn't been altered
- Tests still pass

### 3. Reduced Blast Radius

If an agent makes a mistake or goes off-track, the damage is contained to its designated area.

```bash
# Mistake is contained to the auth module
shai -rw backend/auth -- claude-code
```

### 4. Parallel Workflows

Multiple agents can work simultaneously on different cells without conflicts:

```bash
# Terminal 1: Agent working on auth
shai -rw backend/auth -- claude-code

# Terminal 2: Agent working on payments
shai -rw backend/payments -- gemini-cli

# No conflicts! Each agent has its own cell.
```

### 5. Security

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
# - Cannot access env vars, ~/.aws, or other host resources
#   (unless explicitly exposed via resource sets)
```

**Workspace code visibility is unrestricted.** Agents can see and understand all files in the workspace. This is intentional and important - it allows agents to:
- Understand how components integrate
- Follow existing patterns from other modules
- Avoid breaking interfaces and contracts
- Make informed decisions about their changes

**Write boundaries are the guardrails.** Restricting write access is what prevents agents from making changes beyond their scope, even when they can see opportunities elsewhere in the code.

**Credentials remain protected.** Even though workspace code is readable, environment variables, home directories, and other system resources are isolated unless explicitly exposed through resource sets.

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

- Start with the smallest necessary scope for the task
- Use cellular development even for solo projects (protects against agent overreach)
- Combine related directories when they must change together
- Let agents read the full workspace for context (code visibility is good)
- Think of cells as validation boundaries, not information barriers
- Define cells around components that should be validated as a unit
- Use resource sets to explicitly expose credentials only when needed

### ❌ Don't

- Give agents root-level write access (`-rw .`) - this defeats the guardrails
- Make unrelated directories writable together
- Assume agents need write access everywhere
- Try to "hide" workspace code from agents - that's not the goal
- Forget that boundaries help you validate changes at component edges
- Confuse code visibility (workspace files are readable) with credential access (requires explicit resource sets)

## Next Steps

- Learn about [Resource Sets](../resource-sets) to control what agents can access
- Understand [Apply Rules](../apply-rules) to map paths to resources
- See [Examples](/docs/examples) of cellular development in action
