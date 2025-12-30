---
title: Security
weight: 6
---

Shai provides multiple layers of security to protect your system from unintended agent actions.

## Security Model

### Defense in Depth

Shai uses multiple security layers:

1. **Container Isolation** - Separate namespace from host
2. **Filesystem Restrictions** - Read-only by default
3. **Network Filtering** - HTTP/HTTPS allowlists only
4. **User Isolation** - Non-root execution
5. **Resource Control** - Fine-grained access via resource sets
6. **Ephemeral State** - No persistent changes

### What Shai Protects Against

- ✅ Unintended file modifications
- ✅ Unauthorized network access
- ✅ Credential leakage (via env var mapping)
- ✅ System-wide changes
- ✅ Persistent malicious modifications

### What Shai Doesn't Protect Against

- ❌ Malicious Docker images
- ❌ Resource exhaustion (CPU/memory)
- ❌ Social engineering
- ❌ Intentionally malicious users with config access

## Key Features

### Read-Only Workspace

By default, your entire workspace is mounted read-only:

```bash
shai  # Everything in /src is read-only
```

Only explicitly granted paths are writable:

```bash
shai -rw src/components  # Only src/components is writable
```

### Config File Protection

When workspace root is writable, Shai automatically remounts `.shai/config.yaml` as read-only to prevent sandbox escapes.

### Network Filtering

Three-layer network filtering:

1. **iptables** - Drops unauthorized traffic
2. **dnsmasq** - Blocks DNS for non-allowed domains
3. **tinyproxy** - Filters HTTP/HTTPS requests

View active rules:
```bash
# Inside sandbox
cat /var/log/shai/iptables.out
```

### Credential Handling

Never hardcode secrets:

```yaml
# ❌ Bad - secret in config
vars:
  - source: "sk-1234567890"
    target: API_KEY

# ✅ Good - secret from environment
vars:
  - source: API_KEY
```

### Ephemeral Containers

Containers are automatically removed on exit:
- Changes to writable paths are discarded
- No state persists between sessions
- Fresh environment every time

## Best Practices

### Principle of Least Privilege

Grant minimum necessary access:

```bash
# ❌ Too permissive
shai -rw .

# ✅ Minimal scope
shai -rw src/auth
```

### Resource Set Design

Create focused resource sets:

```yaml
# ✅ Good - specific purpose
resources:
  stripe-api:
    http: [api.stripe.com]
    vars:
      - source: STRIPE_KEY

# ❌ Bad - too broad
resources:
  everything:
    http: ["*"]  # This doesn't work and shouldn't!
```

### Argument Validation

Always use strict regex for remote calls:

```yaml
calls:
  - name: deploy
    command: /usr/local/bin/deploy.sh
    # ✅ Good - strict validation
    allowed-args: '^--env=(staging|production)$'

    # ❌ Bad - allows anything
    # allowed-args: '.*'
```

### Regular Updates

Keep images updated:

```bash
docker pull ghcr.io/colony-2/shai-mega:latest
```

## Threat Scenarios

### Scenario 1: Agent Tries to Modify Root Files

**Attack:**
```bash
# Inside sandbox
rm -rf /
```

**Defense:** Container isolation + file permissions prevent this

### Scenario 2: Agent Tries Unauthorized Network Access

**Attack:**
```bash
curl https://malicious-site.com
```

**Defense:** Network filtering blocks the request (domain not in allowlist)

### Scenario 3: Agent Tries to Escape Container

**Attack:** Various container escape techniques

**Defense:**
- Non-root user (limited capabilities)
- No privileged mode (unless explicitly enabled)
- Read-only filesystem (limited attack surface)

### Scenario 4: Agent Modifies Config

**Attack:**
```bash
echo "privileged: true" >> /src/.shai/config.yaml
```

**Defense:** Config file is remounted read-only when root is writable

## Security Checklist

### ✅ Do

- Use read-only mounts by default
- Map credentials from env vars (never hardcode)
- Use strict argument validation for remote calls
- Review resource sets regularly
- Avoid privileged mode unless absolutely necessary
- Use specific resource sets per path
- Monitor and log remote call usage
- Keep Docker images updated

### ❌ Don't

- Grant write access to entire workspace (`-rw .`)
- Hardcode secrets in config files
- Use permissive argument patterns (`.*`)
- Enable privileged mode without understanding risks
- Share production credentials with development agents
- Skip config file validation
- Trust untrusted Docker images

## Monitoring and Logging

### Network Activity

Check firewall rules:
```bash
# Inside sandbox
cat /var/log/shai/iptables.out
```

### File Access

Monitor which files agents modify:
```bash
# After session ends, check git status
git status
git diff
```

### Remote Calls

Log remote call invocations on the host side in your call scripts.

## Reporting Security Issues

Found a security issue? Report it responsibly:

1. **Don't** open a public GitHub issue
2. **Do** email security@colony2.io (if available) or create a private security advisory
3. Include detailed reproduction steps
4. Allow time for patch before public disclosure

## Further Reading

- [Docker Security Best Practices](https://docs.docker.com/engine/security/)
- [Container Security Guide](https://www.nccgroup.com/us/research-blog/understanding-and-hardening-linux-containers/)
- [OWASP Container Security](https://owasp.org/www-project-docker-top-10/)
