---
title: Configuration Reference
weight: 3
---

Complete reference for `.shai/config.yaml` configuration file.

## Overview

Shai configuration is stored in `.shai/config.yaml` at your workspace root. This file defines:
- Which Docker image to use
- Resource sets (network, mounts, env vars, etc.)
- Apply rules (which paths get which resources)

## Generating a Default Config

You can generate a default config file:

```bash
shai generate
```

This creates `.shai/config.yaml` with sensible defaults.

{{< callout type="info" >}}
**Configuration is optional!** If no config file exists, Shai uses [embedded defaults](https://github.com/colony-2/shai/blob/main/internal/shai/runtime/config/shai.default.yaml) that provide:
- Common package registries (npm, PyPI, etc.)
- Open-source container registries
- Basic network allowlist
{{< /callout >}}

## Config File Location

Shai automatically loads `.shai/config.yaml` from your workspace root.

Override with:
```bash
shai --config /path/to/custom-config.yaml
```

## Loading Behavior

1. Check for `.shai/config.yaml`
2. If not found, use [embedded defaults](https://github.com/colony-2/shai/blob/main/internal/shai/runtime/config/shai.default.yaml)
3. If found, load and validate
4. Fail if config is invalid

## Configuration Sections

{{< cards >}}
  {{< card link="schema" title="Schema Reference" icon="document-text" subtitle="Complete field documentation" >}}
  {{< card link="templates" title="Template Expansion" icon="variable" subtitle="Using dynamic variables" >}}
  {{< card link="example" title="Complete Example" icon="document-text" subtitle="Annotated full config" >}}
{{< /cards >}}

## Quick Example

```yaml
type: shai-sandbox
version: 1
image: ghcr.io/colony-2/shai-mega

resources:
  base-allowlist:
    http:
      - github.com
      - npmjs.org

  frontend-dev:
    http:
      - cdn.jsdelivr.net
    mounts:
      - source: ${{ env.HOME }}/.npm
        target: /home/${{ conf.TARGET_USER }}/.npm
        mode: rw

apply:
  - path: ./
    resources: [base-allowlist]

  - path: frontend
    resources: [frontend-dev]
```

## Next Steps

- [Schema Reference](schema) - Detailed field documentation
- [Template Expansion](templates) - Using variables in config
- [Complete Example](example) - Annotated full configuration
