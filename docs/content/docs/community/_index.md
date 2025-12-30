---
title: Community
weight: 10
---

Get involved with the Shai community.

## Contributing

We welcome contributions! Here's how to get started:

### Types of Contributions

- **Bug reports** - Found an issue? Report it!
- **Feature requests** - Have an idea? Share it!
- **Documentation** - Help improve the docs
- **Code** - Submit pull requests
- **Examples** - Share your config patterns

### Getting Started

1. **Fork the repository**
   ```bash
   git clone https://github.com/colony-2/shai.git
   cd shai
   ```

2. **Build from source**
   ```bash
   go build -o bin/shai ./cmd/shai
   ```

3. **Run tests**
   ```bash
   go test ./...
   ```

4. **Make your changes**
   - Create a feature branch
   - Write tests
   - Update documentation
   - Follow existing code style

5. **Submit a pull request**
   - Describe your changes
   - Link related issues
   - Ensure CI passes

### Development Setup

```bash
# Install Go 1.21+
# Clone repository
git clone https://github.com/colony-2/shai.git
cd shai

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run locally
./bin/shai -rw .
```

### Code of Conduct

Be respectful, inclusive, and constructive. We're all here to build better tools together.

## Support

### Getting Help

1. **Documentation** - Check [docs](/docs) first
2. **GitHub Discussions** - Ask questions
3. **GitHub Issues** - Report bugs
4. **Discord** - Real-time chat (if available)

### Reporting Bugs

[Open an issue](https://github.com/colony-2/shai/issues/new) with:

- **Title** - Clear, concise description
- **Version** - `shai version`
- **Environment** - OS, Docker version
- **Steps to reproduce** - Minimal example
- **Expected behavior** - What should happen
- **Actual behavior** - What actually happened
- **Config** - Sanitized .shai/config.yaml
- **Logs** - Error messages, verbose output

### Feature Requests

[Start a discussion](https://github.com/colony-2/shai/discussions/new) with:

- **Use case** - What problem are you solving?
- **Proposed solution** - How should it work?
- **Alternatives** - Other approaches considered
- **Examples** - Code examples if applicable

## Changelog

### Latest Release

See [GitHub Releases](https://github.com/colony-2/shai/releases) for full changelog.

### Version History

- **v1.0.0** - Initial stable release
  - Core sandboxing features
  - Resource sets and apply rules
  - Remote calls via MCP
  - Official Docker images

## Roadmap

### In Progress

- Enhanced logging and observability
- Windows native support
- Additional security hardening

### Planned

- Web UI for configuration
- Plugin system for extensibility
- Performance optimizations
- Multi-container orchestration

### Under Consideration

- Integration with IDE extensions
- Pre-built resource set library
- Cloud-hosted execution option

Vote on features and suggest new ones in [Discussions](https://github.com/colony-2/shai/discussions).

## Resources

### Official Links

- [GitHub Repository](https://github.com/colony-2/shai)
- [Documentation](/docs)
- [npm Package](https://www.npmjs.com/package/@colony2/shai)
- [Homebrew Tap](https://github.com/colony-2/homebrew-tap)
- [Docker Images](https://github.com/orgs/colony-2/packages)

### Community Resources

- **Examples Repository** - Community-contributed configs
- **Video Tutorials** - Getting started videos
- **Blog Posts** - Deep dives and use cases

### Related Projects

- [Docker](https://docker.com) - Container runtime
- [Claude Code](https://github.com/anthropics/claude-code) - AI coding assistant
- [Devcontainers](https://containers.dev) - Development containers

## License

Shai is released under the [MIT License](https://github.com/colony-2/shai/blob/main/LICENSE).

## Credits

Built by [Colony 2](https://github.com/colony-2).

Special thanks to all [contributors](https://github.com/colony-2/shai/graphs/contributors)!

## Stay Connected

- **GitHub** - Star the repo for updates
- **Twitter** - Follow @colony2_dev (if available)
- **Discord** - Join our community (if available)
- **Newsletter** - Subscribe for release announcements

## Commercial Support

Interested in enterprise support, custom features, or consulting?

Contact: support@colony2.io (if available)
