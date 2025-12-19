# Release Automation Specification

## Overview

This specification outlines the automated release process for `colony2/shai` using GoReleaser Pro to build and distribute artifacts through npm and Homebrew.

GoReleaser Pro handles both npm and Homebrew publishing natively, eliminating the need for manual wrapper scripts or package maintenance. A single configuration file manages cross-platform builds, release notes, and publishing to multiple distribution channels.

## Goals

- Automatically build cross-platform binaries on git tag push
- Publish npm packages as `@colony2/shai`
- Publish Homebrew formula to `colony2/homebrew-tap`
- Generate release notes and GitHub releases
- Support multiple platforms: macOS (amd64/arm64), Linux (amd64/arm64)

## Architecture

```
Git Tag Push (v1.2.3)
    |
    v
GitHub Actions Workflow
    |
    v
GoReleaser Pro
    |
    +-- Build Binaries (cross-platform)
    +-- Create Archives
    +-- Generate Checksums
    +-- Create GitHub Release
    +-- Publish to npm (@colony2/shai)
    +-- Update Homebrew Tap (colony2/homebrew-tap)
```

## Components

### 1. GoReleaser Pro Configuration

**File**: `.goreleaser.yaml`

GoReleaser Pro natively handles both npm and Homebrew publishing. The npm configuration automatically generates a platform-aware package that downloads the correct binary from GitHub releases during installation. No manual npm wrapper code is required.

#### Complete Configuration

```yaml
# .goreleaser.yaml
project_name: shai

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - id: shai
    binary: shai
    main: ./cmd/shai
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: default
    format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    files:
      - README.md
      - LICENSE
      - docs/*

checksum:
  name_template: 'checksums.txt'
  algorithm: sha256

changelog:
  sort: asc
  use: github
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
      - 'typo'
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: Bug Fixes
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999

release:
  github:
    owner: colony-2
    name: shai
  draft: false
  prerelease: auto
  mode: replace
  header: |
    ## Shai {{.Version}}

    Release of Shai sandbox orchestrator.

  footer: |
    **Full Changelog**: https://github.com/colony-2/shai/compare/{{ .PreviousTag }}...{{ .Tag }}

# npm publishing - GoReleaser Pro feature
npm:
  - package:
      name: "@colony2/shai"
      homepage: https://github.com/colony-2/shai
      description: Sandboxing shell for running AI coding agents inside Docker containers
      license: MIT
      author:
        name: Colony 2
        email: noreply@colony2.io
      keywords:
        - sandbox
        - docker
        - ai
        - agent
        - cli
        - security
      repository:
        type: git
        url: git+https://github.com/colony-2/shai.git
      bugs:
        url: https://github.com/colony-2/shai/issues
      engines:
        node: ">=14.0.0"
    publish: true
    registry: https://registry.npmjs.org
    access: public

# Homebrew tap publishing
brews:
  - name: shai
    repository:
      owner: colony-2
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    commit_author:
      name: colony2-bot
      email: bot@colony2.io
    commit_msg_template: "Brew formula update for {{ .ProjectName }} version {{ .Tag }}"
    folder: Formula
    homepage: "https://github.com/colony-2/shai"
    description: "Sandboxing shell for running AI coding agents inside Docker containers"
    license: "MIT"
    test: |
      system "#{bin}/shai", "--version"
    install: |
      bin.install "shai"
```

#### How GoReleaser's npm Publishing Works

GoReleaser Pro automatically:
1. Generates a platform-aware npm package with install scripts
2. Detects the user's OS and architecture during `npm install`
3. Downloads the correct binary from GitHub releases
4. Extracts and installs the binary to `node_modules/.bin/`
5. Makes the `shai` command available globally (with `-g` flag)

This means:
- No manual JavaScript wrapper code needed
- Binary is downloaded on-demand during installation
- Package size on npm is minimal (just metadata + install scripts)
- Users automatically get the correct binary for their platform

### 2. GitHub Actions Workflow

**File**: `.github/workflows/release.yaml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write
  packages: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24.1'

      - name: Run tests
        run: go test -v ./...

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser-pro
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GORELEASER_KEY: ${{ secrets.GORELEASER_KEY }}
          NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

### 3. GitHub Secrets Configuration

Required secrets in `colony-2/shai` repository settings:

| Secret Name | Description | How to Obtain |
|------------|-------------|---------------|
| `GORELEASER_KEY` | GoReleaser Pro license key | From GoReleaser Pro subscription |
| `NPM_TOKEN` | npm authentication token | npm.com -> Access Tokens -> Generate New Token (Automation) |
| `HOMEBREW_TAP_GITHUB_TOKEN` | GitHub PAT for homebrew-tap repo | GitHub Settings -> Developer Settings -> Personal Access Tokens -> Fine-grained token with `contents: write` on `colony-2/homebrew-tap` |
| `GITHUB_TOKEN` | Default GitHub Actions token | Automatically provided by GitHub Actions |

### 4. Homebrew Tap Repository

**Repository**: `colony-2/homebrew-tap`

#### Structure
```
homebrew-tap/
├── Formula/
│   └── shai.rb          # Generated by GoReleaser
└── README.md
```

#### Initial Setup

1. Create repository at `https://github.com/colony-2/homebrew-tap`
2. Initialize with README:

```markdown
# Colony 2 Homebrew Tap

Homebrew formulae for Colony 2 tools.

## Installation

```bash
brew tap colony-2/tap
brew install shai
```

## Available Formulae

- `shai` - Sandboxing shell for running AI coding agents inside Docker containers
```

3. GoReleaser will automatically create/update `Formula/shai.rb` on releases

### 5. Version Management

#### Versioning Scheme

Follow Semantic Versioning (semver): `vMAJOR.MINOR.PATCH`

- `v1.0.0` - Initial stable release
- `v1.1.0` - New features, backward compatible
- `v1.0.1` - Bug fixes
- `v2.0.0` - Breaking changes

#### Tagging Process

```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0: Initial stable release"

# Push tag to trigger release
git push origin v1.0.0
```

#### Pre-releases

Use suffix for pre-releases:
- `v1.0.0-alpha.1`
- `v1.0.0-beta.1`
- `v1.0.0-rc.1`

GoReleaser will automatically mark these as pre-releases on GitHub.

### 6. Installation Methods for End Users

#### Homebrew (macOS/Linux)
```bash
brew tap colony-2/tap
brew install shai
shai --version
```

#### npm (All platforms)
```bash
npm install -g @colony2/shai
shai --version
```

#### Direct Download
```bash
# Download from GitHub releases
curl -LO https://github.com/colony-2/shai/releases/download/v1.0.0/shai_Linux_x86_64.tar.gz
tar xzf shai_Linux_x86_64.tar.gz
sudo mv shai /usr/local/bin/
```

## Implementation Checklist

### Prerequisites
- [ ] GoReleaser Pro subscription and license key
- [ ] npm account with organization `@colony2`
- [ ] GitHub repository `colony-2/homebrew-tap` created
- [ ] GitHub secrets configured in `colony-2/shai`

### Initial Setup
- [ ] Create `.goreleaser.yaml` configuration
- [ ] Create `.github/workflows/release.yaml`
- [ ] Update `cmd/shai/main.go` to include version variables
- [ ] Add `LICENSE` file if not present
- [ ] Test GoReleaser locally: `goreleaser release --snapshot --clean`

### First Release
- [ ] Commit all configuration files
- [ ] Tag first release: `git tag -a v1.0.0 -m "Release v1.0.0"`
- [ ] Push tag: `git push origin v1.0.0`
- [ ] Monitor GitHub Actions workflow
- [ ] Verify GitHub release created
- [ ] Verify npm package published: `npm view @colony2/shai`
- [ ] Verify Homebrew formula: `brew install colony-2/tap/shai`

### Documentation Updates
- [ ] Update README.md with installation instructions
- [ ] Add badge for latest release
- [ ] Document release process for maintainers
- [ ] Add CHANGELOG.md for release notes

## Testing

### Local Testing

```bash
# Test build without publishing
goreleaser release --snapshot --clean --skip=publish

# Check generated artifacts in dist/
ls -lh dist/

# Test npm package locally (GoReleaser generates this automatically)
cd dist/@colony2-shai_*
npm pack
npm install -g colony2-shai-*.tgz
shai --version
```

### Pre-release Testing

```bash
# Create pre-release tag
git tag -a v1.0.0-rc.1 -m "Release candidate 1"
git push origin v1.0.0-rc.1

# Verify pre-release flag on GitHub
# Install and test
npm install -g @colony2/shai@1.0.0-rc.1
brew install colony-2/tap/shai@1.0.0-rc.1
```

## Troubleshooting

### GoReleaser Errors

**Issue**: `invalid GoReleaser Pro license`
- Verify `GORELEASER_KEY` secret is set correctly
- Check license is valid and not expired

**Issue**: `npm publish failed`
- Verify `NPM_TOKEN` has automation permissions
- Check package name `@colony2/shai` is not already taken
- Verify organization `@colony2` exists on npm

**Issue**: `Homebrew tap update failed`
- Verify `HOMEBREW_TAP_GITHUB_TOKEN` has write access to `colony-2/homebrew-tap`
- Check repository exists and is accessible
- Ensure token has `contents: write` permission

### Build Failures

**Issue**: Tests fail during release
- Fix failing tests before tagging
- Or use `--skip=validate` flag (not recommended)

**Issue**: Cross-compilation errors
- Ensure CGO is disabled: `CGO_ENABLED=0`
- Check all dependencies support target platforms

## Maintenance

### Regular Tasks

1. **Monthly**: Review and update dependencies
   ```bash
   go get -u ./...
   go mod tidy
   ```

2. **Per Release**: Update CHANGELOG.md with notable changes

3. **Annually**: Review and update GoReleaser configuration for new features

### Monitoring

- Monitor npm downloads: https://www.npmjs.com/package/@colony2/shai
- Monitor Homebrew analytics: `brew info shai`
- Track GitHub release downloads
- Review GitHub Actions workflow runs for failures

## References

- GoReleaser Pro Documentation: https://goreleaser.com/pro/
- npm Publishing Guide: https://docs.npmjs.com/packages-and-modules/contributing-packages-to-the-registry
- Homebrew Formula Cookbook: https://docs.brew.sh/Formula-Cookbook
- GitHub Actions Documentation: https://docs.github.com/en/actions
- Semantic Versioning: https://semver.org/

## Support

For issues with the release process:
- GoReleaser: https://github.com/goreleaser/goreleaser/discussions
- npm: https://docs.npmjs.com/support
- Homebrew: https://github.com/Homebrew/homebrew-core/issues
