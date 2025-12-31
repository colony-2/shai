---
title: shai
layout: hextra-home
---

<style>
  .rainbow-text {
    /* Create the rainbow gradient */
    background: linear-gradient(to right, #ef5350, #f48fb1, #7e57c2, #2196f3, #26c6da, #43a047, #eeff41, #f9a825, #ff5722);
    
    /* Clip the background to the text */
    -webkit-background-clip: text;
    background-clip: text;
    
    /* Make the actual text transparent so the background shows through */
    color: transparent;
    
  }
</style>

{{< hextra/hero-badge >}}
  <div class="hx:w-2 hx:h-2 hx:rounded-full hx:bg-primary-400"></div>
  <span>Free and open source</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx:mt-6 hx:mb-6">
{{< hextra/hero-headline >}}
  Sandboxing Shell&nbsp;<br class="hx:sm:block hx:hidden" />for AI Coding Agents
{{< /hextra/hero-headline >}}
</div>

<div>
{{< hextra/hero-subtitle >}}
  Let AI agents run free... &nbsp;<br class="hx:sm:block hx:hidden" />without letting them <span class="rainbow-text">run wild</span>.
{{< /hextra/hero-subtitle >}}
</div>



  ```bash {linenos=false}
  npm install -g @colony2/shai     
  ```


<div class="hx:mb-6 hx:mt-6">
{{< hextra/hero-button text="Get Started" link="docs/quick-start" >}}
{{< hextra/hero-button text="View on GitHub" link="https://github.com/colony-2/shai" >}}
</div>

<div class="hx:mt-6"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Secure by Default"
    subtitle="Read-only workspace, network filtering, and container isolation protect your system from unintended agent actions."
    style="background: radial-gradient(ellipse at 50% 80%,rgba(194,97,254,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Built for Cellular Development"
    subtitle="Establish clear boundaries and guardrails. Agents can read all workspace code for context but only modify designated areas, preventing scope creep and overreach."
    style="background: radial-gradient(ellipse at 50% 80%,rgba(142,53,74,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Composable Security Policies"
    subtitle="Resource sets, application rules, and selective elevation provide fine-grained access control."
    style="background: radial-gradient(ellipse at 50% 80%,rgba(221,210,59,0.15),hsla(0,0%,100%,0));"
  >}}
  {{< hextra/feature-card
    title="Works with Any Tool"
    subtitle="Compatible with Claude Code, Codex, Gemini CLI and any other CLI-based AI coding agent."
  >}}
  {{< hextra/feature-card
    title="Network Filtering"
    subtitle="HTTP/HTTPS allowlists control exactly which APIs and services your agents can access. Block unwanted connections by default."
  >}}
  {{< hextra/feature-card
    title="Ephemeral Containers"
    subtitle="Each session runs in a fresh container. No persistent modifications to your system. Exit and it's gone."
  >}}
{{< /hextra/feature-grid >}}


[//]: # ()
[//]: # ({{< cards cols="4">}})

[//]: # (  {{< card link="docs/concepts" title="Core Concepts" icon="book-open" >}})

[//]: # (  {{< card link="docs/configuration" title="Configuration Reference" icon="cog" >}})

[//]: # (  {{< card link="docs/examples" title="Examples" icon="document-text" >}})

[//]: # (  {{< card link="docs/security" title="Security" icon="shield-check" >}})

[//]: # ({{< /cards >}})
