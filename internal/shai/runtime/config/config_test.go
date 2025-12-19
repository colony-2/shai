package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, dir, contents string) string {
	t.Helper()
	path := filepath.Join(dir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}

func TestLoadConfigHappyPath(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
image: ghcr.io/example/image:latest
resources:
  global:
    vars:
      - source: XYZ
        target: AXYZ
    mounts:
      - source: ${{ env.HOME }}/.ssh
        target: /home/${{ conf.TARGET_USER }}/.ssh
        mode: rw
    calls:
      - name: git-sync
        command: git pull
    http:
      - googleapis.com
    ports:
      - host: github.com
        port: 443
  dir-only:
    calls:
      - name: cache-rm
        command: ops/cache_rm.sh
apply:
  - path: ./
    resources:
      - global
  - path: ./dir
    resources:
      - global
      - dir-only
`)

	env := map[string]string{"HOME": "/Users/test", "XYZ": "ignored"}
	cfg, err := Load(path, env, map[string]string{})
	require.NoError(t, err)

	assert.Equal(t, "shai", cfg.User)
	assert.Equal(t, "/src", cfg.Workspace)
	res := cfg.Resources["global"]
	require.NotNil(t, res)
	require.Len(t, res.Mounts, 1)
	assert.Equal(t, "/Users/test/.ssh", res.Mounts[0].Source)
	assert.Equal(t, "/home/shai/.ssh", res.Mounts[0].Target)
	assert.Equal(t, "rw", res.Mounts[0].Mode)

	rootResources := cfg.ResourcesForPath("docs")
	require.Len(t, rootResources, 1)
	assert.Equal(t, "global", rootResources[0].Name)

	dirResources := cfg.ResourcesForPath("dir/subdir/file")
	require.Len(t, dirResources, 2)
	assert.Equal(t, []string{"global", "dir-only"}, []string{dirResources[0].Name, dirResources[1].Name})
}

func TestLoadConfigUnknownResource(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
image: ghcr.io/example/image:latest
resources:
  base: {}
apply:
  - path: ./
    resources: [base, missing]
`)
	_, err := Load(path, map[string]string{}, map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource")
}

func TestLoadConfigDuplicateCall(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
image: ghcr.io/example/image:latest
resources:
  first:
    calls:
      - name: git-sync
        command: git pull
  second:
    calls:
      - name: git-sync
        command: git pull --rebase
apply:
  - path: ./
    resources:
      - first
      - second
`)
	_, err := Load(path, map[string]string{}, map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "call \"git-sync\"")
}

func TestApplyRulesRequireSegmentMatch(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
image: ghcr.io/example/image:latest
resources:
  bar-only: {}
apply:
  - path: ./bar
    resources: [bar-only]
`)
	cfg, err := Load(path, map[string]string{}, map[string]string{})
	require.NoError(t, err)

	require.Len(t, cfg.ResourcesForPath("bar"), 1)
	require.Len(t, cfg.ResourcesForPath("bar/baz"), 1)
	assert.Empty(t, cfg.ResourcesForPath("bar-boo"))
	assert.Empty(t, cfg.ResourcesForPath("barboo/qux"))
}

func TestApplyRootImageOverrideDisallowed(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
image: ghcr.io/example/image:latest
resources:
  base: {}
apply:
  - path: ./
    image: ghcr.io/foo/bar:latest
    resources: [base]
`)
	_, err := Load(path, map[string]string{}, map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot override image")
}

func TestImageForPathPrefersMostSpecific(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
image: ghcr.io/example/image:latest
resources:
  base: {}
apply:
  - path: ./
    resources: [base]
  - path: ./bar
    image: ghcr.io/parent/img:latest
    resources: [base]
  - path: ./bar/baz
    image: ghcr.io/child/img:latest
    resources: [base]
`)
	cfg, err := Load(path, map[string]string{}, map[string]string{})
	require.NoError(t, err)

	img, ok := cfg.ImageForPath("bar/baz/qux")
	require.True(t, ok)
	assert.Equal(t, "ghcr.io/child/img:latest", img)

	img, ok = cfg.ImageForPath("bar/qux")
	require.True(t, ok)
	assert.Equal(t, "ghcr.io/parent/img:latest", img)
}

func TestTemplateMissingVar(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
user: ${{ env.USERNAME }}
workspace: /src
resources:
  base: {}
apply:
  - path: ./
    resources: [base]
`)
	_, err := Load(path, map[string]string{}, map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "env \"USERNAME\" not found")
}

func TestLoadOrDefaultFallsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".shai", "config.yaml")

	// Provide HOME environment variable for template expansion in default config
	env := map[string]string{"HOME": "/home/testuser"}
	cfg, usedDefault, err := LoadOrDefault(path, env, map[string]string{})
	require.NoError(t, err)
	assert.True(t, usedDefault)
	require.NotNil(t, cfg)

	res := cfg.Resources["shai-default-allow"]
	require.NotNil(t, res, "default config should define shai-default-allow resource")

	resolved := cfg.ResolveResources([]string{"."})
	names := make([]string, len(resolved))
	for i, r := range resolved {
		names[i] = r.Name
	}
	assert.Contains(t, names, "shai-default-allow")
}

func TestLoadOrDefaultUsesFileWhenPresent(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
type: shai-sandbox
version: 1
image: example
user: dev
workspace: /src
resources:
  base: {}
apply:
  - path: ./
    resources: [base]
`)

	cfg, usedDefault, err := LoadOrDefault(path, map[string]string{}, map[string]string{})
	require.NoError(t, err)
	assert.False(t, usedDefault)
	require.NotNil(t, cfg)
	assert.Equal(t, "dev", cfg.User)
}
