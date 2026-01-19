package shai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/colony-2/shai/internal/shai/runtime/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderedResourcePaths(t *testing.T) {
	paths := orderedResourcePaths([]string{"src", ".", "src", "pkg"})
	require.Equal(t, []string{".", "src", "pkg"}, paths)
}

func TestCallEntriesFromResources(t *testing.T) {
	resources := []*config.ResolvedResource{
		{
			Name: "global",
			Spec: &config.ResourceSet{
				Calls: []config.Call{
					{Name: "git-sync", Command: "git pull --rebase"},
				},
			},
		},
		{
			Name: "feature",
			Spec: &config.ResourceSet{
				Calls: []config.Call{
					{Name: "git-sync", Command: "git pull"},
					{Name: "deploy", Command: "./scripts/deploy.sh", AllowedArgs: "^--env=(dev|prod)$"},
				},
			},
		},
	}

	entries, err := callEntriesFromResources(resources)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
		if e.Name == "deploy" {
			assert.NoError(t, e.ValidateArgs("--env=dev"))
			assert.Error(t, e.ValidateArgs("--env=qa"))
		}
	}
	assert.ElementsMatch(t, []string{"git-sync", "deploy"}, names)
}

func TestResolvedResourcesWithExtraSets(t *testing.T) {
	cfg := loadTestConfig(t, `
type: shai-sandbox
version: 1
image: example
user: dev
workspace: /src
resources:
  base:
    vars:
      - source: FOO
        target: BAR
  opt:
    http: ["example.com"]
  another: {}
apply:
  - path: ./
    resources: [base]
`)

	resources, names, image, err := resolvedResources(cfg, []string{"."}, []string{"opt", "base", "another"}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"opt", "base", "another"}, names)
	require.Len(t, resources, 3)
	assert.Equal(t, "", image)
}

func TestResolvedResourcesWithAppendSet(t *testing.T) {
	cfg := loadTestConfig(t, `
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

	appendSet := &config.ResourceSet{
		HTTP: []string{"example.com"},
	}
	resources, names, _, err := resolvedResources(cfg, []string{"."}, []string{"base"}, nil, appendSet)
	require.NoError(t, err)
	require.Equal(t, []string{"base"}, names)
	require.Len(t, resources, 2)
	assert.Equal(t, []string{"example.com"}, resources[1].Spec.HTTP)
}

func TestResolvedResourcesWithPrependSet(t *testing.T) {
	cfg := loadTestConfig(t, `
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

	prependSet := &config.ResourceSet{
		HTTP: []string{"prepend.example"},
	}
	resources, names, _, err := resolvedResources(cfg, []string{"."}, []string{"base"}, prependSet, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"base"}, names)
	require.Len(t, resources, 2)
	assert.Equal(t, []string{"prepend.example"}, resources[0].Spec.HTTP)
}

func TestResolvedResourcesUnknownSet(t *testing.T) {
	cfg := loadTestConfig(t, `
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

	_, _, _, err := resolvedResources(cfg, []string{"."}, []string{"missing", "base"}, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestResolvedResourcesImageOverrideByOverlay(t *testing.T) {
	cfg := loadTestConfig(t, `
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
  - path: ./foo
    resources: [base]
    image: foo-image
  - path: ./bar
    resources: [base]
    image: bar-image
`)

	_, _, image, err := resolvedResources(cfg, []string{"bar", "foo"}, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "bar-image", image)

	_, _, image, err = resolvedResources(cfg, []string{"foo", "bar"}, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "foo-image", image)
}

func TestResolvedResourcesImageOverridePrefersSpecificPath(t *testing.T) {
	cfg := loadTestConfig(t, `
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
  - path: ./bar
    resources: [base]
    image: bar-image
  - path: ./bar/baz
    resources: [base]
    image: baz-image
`)

	_, _, image, err := resolvedResources(cfg, []string{"bar/baz"}, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "baz-image", image)

	_, _, image, err = resolvedResources(cfg, []string{"bar/qux"}, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "bar-image", image)
}

func loadTestConfig(t *testing.T, contents string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".shai", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(contents)+"\n"), 0o644))
	cfg, err := config.Load(path, map[string]string{"FOO": "bar"}, map[string]string{})
	require.NoError(t, err)
	return cfg
}
