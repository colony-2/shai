package shai

import (
	"os"
	"path/filepath"
	"testing"

	configpkg "github.com/colony-2/shai/internal/shai/runtime/config"
	"github.com/stretchr/testify/require"
)

func TestBuildBootstrapArgsIncludesResources(t *testing.T) {
	runner := &EphemeralRunner{
		shaiConfig: &configpkg.Config{
			User:      "shai",
			Workspace: "/src",
		},
		resources: []*configpkg.ResolvedResource{
			{
				Name: "base",
				Spec: &configpkg.ResourceSet{
					Vars: []configpkg.VarMapping{
						{Source: "TOKEN", Target: "INSIDE_TOKEN"},
					},
					HTTP: []string{"example.com"},
					Ports: []configpkg.Port{
						{Host: "github.com", Port: 443},
					},
				},
			},
		},
		hostEnv: map[string]string{
			"TOKEN": "super-secret",
		},
		config: EphemeralConfig{
			PostSetupExec: &ExecSpec{
				Command: []string{"/bin/bash", "-lc", "echo hi"},
				Env: map[string]string{
					"FOO": "bar",
				},
				UseTTY: false,
			},
			Verbose: true,
		},
	}

	args, err := runner.buildBootstrapArgs()
	require.NoError(t, err)

	require.Equal(t, []string{
		"--version", "1",
		"--user", "shai",
		"--workspace", "/src",
		"--rm", "true",
		"--exec-env", "INSIDE_TOKEN=super-secret",
		"--exec-cmd", "/bin/bash",
		"--exec-cmd", "-lc",
		"--exec-cmd", "echo hi",
		"--exec-env", "FOO=bar",
		"--http-allow", "example.com",
		"--port-allow", "github.com:443",
		"--verbose",
	}, args)
}

func TestBuildBootstrapArgsMissingEnvFails(t *testing.T) {
	runner := &EphemeralRunner{
		shaiConfig: &configpkg.Config{
			User:      "shai",
			Workspace: "/src",
		},
		resources: []*configpkg.ResolvedResource{
			{
				Name: "base",
				Spec: &configpkg.ResourceSet{
					Vars: []configpkg.VarMapping{
						{Source: "MISSING", Target: "INSIDE"},
					},
				},
			},
		},
		hostEnv: map[string]string{},
	}

	_, err := runner.buildBootstrapArgs()
	require.Error(t, err)
	require.Contains(t, err.Error(), "host env \"MISSING\" not set")
}

func TestChooseImagePrecedence(t *testing.T) {
	img, src := chooseImage("base", "cli-override", "apply-override")
	require.Equal(t, "cli-override", img)
	require.Equal(t, "cli", src)

	img, src = chooseImage("base", "", "apply-override")
	require.Equal(t, "apply-override", img)
	require.Equal(t, "apply", src)

	img, src = chooseImage("base", "  ", "")
	require.Equal(t, "base", img)
	require.Equal(t, "", src)
}

func TestResourceMountsSkipMissingPaths(t *testing.T) {
	tDir := t.TempDir()
	existing := filepath.Join(tDir, "existing")
	err := os.Mkdir(existing, 0o755)
	require.NoError(t, err)

	runner := &EphemeralRunner{
		config: EphemeralConfig{
			WorkingDir: tDir,
		},
		resources: []*configpkg.ResolvedResource{
			{
				Name: "set",
				Spec: &configpkg.ResourceSet{
					Mounts: []configpkg.Mount{
						{Source: "existing", Target: "/existing", Mode: "rw"},
						{Source: "missing", Target: "/missing", Mode: "ro"},
					},
				},
			},
		},
	}

	mounts, err := runner.resourceMounts()
	require.NoError(t, err)
	require.Len(t, mounts, 1)
	require.Equal(t, existing, mounts[0].Source)
	require.Equal(t, "/existing", mounts[0].Target)
	require.False(t, mounts[0].ReadOnly)
}

func TestEffectiveWorkspaceSinglePath(t *testing.T) {
	require.Equal(t, "/src/cmd", effectiveWorkspace("/src", []string{"cmd"}))
	require.Equal(t, "/src", effectiveWorkspace("/src", []string{"."}))
}

func TestEffectiveWorkspaceDefaults(t *testing.T) {
	require.Equal(t, "/src", effectiveWorkspace("/src", nil))
	require.Equal(t, "/src", effectiveWorkspace("/src", []string{"one", "two"}))
	require.Equal(t, "/src", effectiveWorkspace("/src", []string{"/abs"}))
}

func TestEffectiveWorkspaceWithDotPrefix(t *testing.T) {
	require.Equal(t, "/src/cmd", effectiveWorkspace("/src", []string{"./cmd"}))
}

func TestBuildDockerConfigsInjectsHostIDs(t *testing.T) {
	tDir := t.TempDir()
	mountBuilder, err := NewMountBuilder(tDir, nil)
	require.NoError(t, err)

	runner := &EphemeralRunner{
		config: EphemeralConfig{
			WorkingDir: tDir,
		},
		shaiConfig: &configpkg.Config{
			User:      "shai",
			Workspace: "/src",
		},
		mountBuilder: mountBuilder,
		image:        "example",
		hostEnv:      map[string]string{},
		hostUID:      "1234",
		hostGID:      "5678",
	}

	cfg, _, err := runner.buildDockerConfigs(false, "sandbox-test")
	require.NoError(t, err)
	require.Contains(t, cfg.Env, "DEV_UID=1234")
	require.Contains(t, cfg.Env, "DEV_GID=5678")
}
