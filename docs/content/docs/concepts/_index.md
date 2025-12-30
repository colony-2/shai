---
title: Core Concepts
weight: 2
---

Understanding these core concepts will help you use Shai effectively.

## Overview

Shai is built around five key concepts:

1. **[Cellular Development](cellular-development)** - Constraining agents to specific components
2. **[Resource Sets](resource-sets)** - Defining collections of allowed resources
3. **[Apply Rules](apply-rules)** - Mapping workspace paths to resource sets
4. **[Selective Elevation](selective-elevation)** - Controlled host-side operations
5. **[How Shai Works](how-it-works)** - Understanding the architecture

## Quick Mental Model

Think of Shai as a bouncer for your AI agents:

- **Cellular Development**: Only let the agent into certain rooms (directories)
- **Resource Sets**: Define what's available in each room (network, files, tools)
- **Apply Rules**: Decide which rooms get which resources
- **Selective Elevation**: Provide a phone to call outside when needed
- **Architecture**: How the bouncer enforces all these rules

{{< cards >}}
  {{< card link="cellular-development" title="Cellular Development" icon="cube" >}}
  {{< card link="resource-sets" title="Resource Sets" icon="cube" >}}
  {{< card link="apply-rules" title="Apply Rules" icon="document-text" >}}
  {{< card link="selective-elevation" title="Selective Elevation" icon="arrow-up" >}}
  {{< card link="how-it-works" title="How Shai Works" icon="cog" >}}
{{< /cards >}}
