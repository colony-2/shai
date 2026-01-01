---
title: Community
weight: 10
---

Get involved with the Shai community.

## Contributing

We welcome contributions! Here's how to get started:

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
   go test ./... --tags=integration
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

### Code of Conduct

Be respectful, inclusive, and constructive. We're all here to build better tools together.

## Support

### Getting Help

1. [Documentation](/docs)
2. [GitHub Issues](https://github.com/colony-2/shai/issues)

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

[Open an issue](https://github.com/colony-2/shai/issues/new) with:

- **Use case** - What problem are you solving?
- **Proposed solution** - How should it work?
- **Alternatives** - Other approaches considered
- **Examples** - Code examples if applicable

## Related Projects

- [Docker](https://docker.com) - Container runtime
- [Claude Code](https://github.com/anthropics/claude-code) - AI coding assistant
- [Devcontainers](https://containers.dev) - Development containers

## License

Shai is released under the [MIT License](https://github.com/colony-2/shai/blob/main/LICENSE).

## Credits

Built by [Colony 2](https://colony2.com).

