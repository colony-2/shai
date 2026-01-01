---
title: Troubleshooting
weight: 9
---

Common issues and solutions.

## Installation Issues

### Docker Not Running

**Error:**
```
Error: Cannot connect to the Docker daemon
```

**Solution:**
Start Docker Desktop or the Docker daemon:

```bash
# macOS
open -a Docker

# Linux (systemd)
sudo systemctl start docker

# Check status
docker ps
```

### Permission Denied

**Error:**
```
permission denied while trying to connect to the Docker daemon socket
```

**Solution:**
Add your user to the docker group:

```bash
sudo usermod -aG docker $USER
```

Then log out and back in.

### npm Install Fails

**Error:**
```
npm ERR! code EACCES
```

**Solution:**
Don't use sudo with npm global installs. Fix npm permissions:

```bash
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
npm install -g @colony2/shai
```

## Runtime Issues

### Path Not Writable

**Error:**
```
Error: path does not exist: src/components
```

**Solution:**
The path must exist before running shai:

```bash
# Check path exists
ls src/components

# Create if missing
mkdir -p src/components

# Then run shai
shai -rw src/components
```

### Config Validation Error

**Error:**
```
Error: Failed to load config
Cause: Template variable not found: env.API_KEY
```

**Solution:**
Set the required environment variable:

```bash
export API_KEY=your-key-here
shai -rw src
```

Or remove the reference from config if not needed.

### Network Access Blocked

**Error:**
```
curl: (6) Could not resolve host: example.com
```

**Solution:**
Add the domain to your resource set's HTTP allowlist:

```yaml
resources:
  my-tools:
    http:
      - example.com
```

View active rules inside the sandbox:
```bash
cat /var/log/shai/iptables.out
```

### Image Pull Fails

**Error:**
```
Error: Failed to pull image ghcr.io/colony-2/shai-mega:latest
```

**Solutions:**

1. Check network connectivity:
   ```bash
   ping github.com
   ```

2. Authenticate if using private images:
   ```bash
   echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
   ```

3. Use a different image:
   ```bash
   shai --image ghcr.io/colony-2/shai-base:latest
   ```

### Container Won't Start

**Error:**
```
Error: Container exited with code 1
```

**Debug:**

1. Check Docker logs:
   ```bash
   docker ps -a  # Find container ID
   docker logs <container-id>
   ```

2. Run with verbose:
   ```bash
   shai -rw src --verbose
   ```

3. Verify image works:
   ```bash
   docker run --rm ghcr.io/colony-2/shai-mega:latest which supervisord
   ```

## Configuration Issues

### Resource Set Not Applied

**Problem:** Resource set doesn't seem to work

**Debug:**

1. Use verbose mode:
   ```bash
   shai -rw src --verbose
   ```

2. Check apply rules match your path:
   ```yaml
   apply:
     - path: src  # Matches src/**/*
       resources: [my-tools]
   ```

3. Verify resource set name is correct (case-sensitive)

### Call Not Available

**Error:**
```
shai-remote: command not found: my-call
```

**Solutions:**

1. Check call is defined in a resource set that's applied
2. Verify call name (case-sensitive)
3. List available calls:
   ```bash
   shai-remote --list
   ```

### Template Expansion Fails

**Error:**
```
Template variable not found: vars.TAG
```

**Solution:**
Provide the variable:

```bash
shai --var TAG=v1.0.0 -rw src
```

## Performance Issues

### Slow Startup

**Problem:** Shai takes a long time to start

**Solutions:**

1. Pre-pull the image:
   ```bash
   docker pull ghcr.io/colony-2/shai-mega:latest
   ```

2. Use shai-base instead of shai-mega:
   ```yaml
   image: ghcr.io/colony-2/shai-base:latest
   ```

3. Build a custom minimal image

### High Memory Usage

**Problem:** Container uses too much memory

**Solutions:**

1. Limit container memory:
   ```bash
   docker run --memory=2g ...
   ```

2. Use shai-base (lighter image)

3. Check what's running:
   ```bash
   # Inside sandbox
   htop
   ```

### Disk Space Issues

**Problem:** Running out of disk space

**Solutions:**

1. Clean up old containers:
   ```bash
   docker container prune
   ```

2. Remove unused images:
   ```bash
   docker image prune -a
   ```

3. Check Docker disk usage:
   ```bash
   docker system df
   ```

## Agent-Specific Issues

### Claude Code Sandbox Warning

**Warning:**
```
Warning: Running outside of sandbox
```

**Solution:**
Use the bypass flag since Shai provides sandboxing:

```bash
shai -rw src -- claude --dangerously-bypass-approvals-and-sandbox
```

### Agent Can't Install Packages

**Problem:** npm/pip install fails

**Solutions:**

1. Add package registry to HTTP allowlist:
   ```yaml
   http:
     - npmjs.org
     - registry.npmjs.org
     - pypi.org
   ```

2. Mount package cache:
   ```yaml
   mounts:
     - source: ${{ env.HOME }}/.npm
       target: /home/${{ conf.TARGET_USER }}/.npm
       mode: rw
   ```

### Agent Can't Access Git

**Problem:** git clone/push fails

**Solutions:**

1. Add git SSH access:
   ```yaml
   ports:
     - host: github.com
       port: 22
   ```

2. Mount SSH keys:
   ```yaml
   mounts:
     - source: ${{ env.HOME }}/.ssh
       target: /home/${{ conf.TARGET_USER }}/.ssh
       mode: ro
   ```

3. Add HTTPS access:
   ```yaml
   http:
     - github.com
   ```

## Debugging Techniques

### Enable Verbose Mode

```bash
shai -rw src --verbose
```

Shows:
- Bootstrap script
- Resource resolution
- Network rules
- Mount points

### Inspect Container

```bash
# Start sandbox
shai -rw src

# In another terminal
docker ps  # Find container name (shai-*)
docker exec -it <container-name> bash

# Inside container
cat /var/log/shai/iptables.out
ps aux
env
```

### Test Network Filtering

```bash
shai -- bash -c "
  # Should work (if in allowlist)
  curl -I https://github.com

  # Should fail
  curl -I https://example.com

  # Check DNS
  nslookup github.com
"
```

### Verify Mounts

```bash
shai -rw src --verbose -- mount | grep /src
```

## Getting Help

### Check Documentation

- [Quick Start](/docs/quick-start)
- [Configuration Reference](/docs/configuration)
- [Examples](/docs/examples)

### Report Issues

[GitHub Issues](https://github.com/colony-2/shai/issues)

Include:
- Shai version (`shai version`)
- Operating system
- Docker version (`docker version`)
- Config file (sanitized)
- Full error message
- Steps to reproduce

## FAQ

**Q: Can I run Docker inside Shai?**

A: Yes, but requires privileged mode. You can review the [.shai/config.yaml](https://github.com/colony-2/shai/blob/main/.shai/config.yaml) in the shai repo to see how it runs DIND for tests.

```yaml
resources:
  docker:
    mounts:
      - source: /var/run/docker.sock
        target: /var/run/docker.sock
        mode: rw
    options:
      privileged: true
```

**Q: Can I use Shai on Windows?**

A: Probably, but not officially supported. We're [still working on some path resolution issues](https://github.com/colony-2/shai/pull/4).

**Q: How do I share data between sessions?**

A: Mount a host directory:

```yaml
mounts:
  - source: /path/to/shared
    target: /shared
    mode: rw
```

**Q: Why is my workspace read-only?**

A: This is intentional! Use `-rw` to grant write access:

```bash
shai -rw src
```

**Q: Can I modify .shai/config.yaml?**

A: On the host, yes. Not inside the container. It's automatically remounted read-only to avoid a agent escalating it's own access.
