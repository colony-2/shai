package shai

import (
	"path/filepath"
	"testing"
)

func TestLoadSandboxConfigDefaults(t *testing.T) {
	cfg, err := LoadSandboxConfig("")
	if err != nil {
		t.Fatalf("LoadSandboxConfig errored: %v", err)
	}
	if cfg.WorkingDir == "" {
		t.Fatalf("expected working dir to be populated")
	}
	wantConfig := filepath.Join(cfg.WorkingDir, DefaultConfigRelPath)
	if cfg.ConfigFile != wantConfig {
		t.Fatalf("expected default config file %q, got %q", wantConfig, cfg.ConfigFile)
	}
}

func TestRuntimeConfigConversion(t *testing.T) {
	cfg := SandboxConfig{
		WorkingDir: "/workspace",
		ConfigFile: "/workspace/" + DefaultConfigRelPath,
		ReadWritePaths: []string{
			".",
		},
		PrependResourceSet: &ResourceSet{
			HTTP: []string{"prepend.example"},
		},
		AppendResourceSet: &ResourceSet{
			HTTP: []string{"append.example"},
		},
		PostSetupExec: &SandboxExec{
			Command: []string{"echo", "hi"},
			Env: map[string]string{
				"FOO": "bar",
			},
			Workdir: "/src",
			UseTTY:  true,
		},
	}

	rc, err := cfg.runtimeConfig()
	if err != nil {
		t.Fatalf("runtime config error: %v", err)
	}
	if rc.WorkingDir != cfg.WorkingDir {
		t.Fatalf("runtime working dir mismatch: %q != %q", rc.WorkingDir, cfg.WorkingDir)
	}
	if rc.PostSetupExec == nil {
		t.Fatalf("expected post setup exec to be copied")
	}
	if got, want := rc.PostSetupExec.Command, cfg.PostSetupExec.Command; len(got) != len(want) {
		t.Fatalf("command mismatch, got %v want %v", got, want)
	}
	if rc.PostSetupExec.UseTTY != cfg.PostSetupExec.UseTTY {
		t.Fatalf("useTTY mismatch")
	}
	if rc.PrependResourceSet == nil || len(rc.PrependResourceSet.HTTP) != 1 {
		t.Fatalf("expected prepend resource set to be converted")
	}
	if rc.AppendResourceSet == nil || len(rc.AppendResourceSet.HTTP) != 1 {
		t.Fatalf("expected append resource set to be converted")
	}
}

func TestRuntimeConfigResourceSetValidation(t *testing.T) {
	cfg := SandboxConfig{
		WorkingDir: "/workspace",
		ConfigFile: "/workspace/" + DefaultConfigRelPath,
		PrependResourceSet: &ResourceSet{
			Mounts: []Mount{
				{Source: "/tmp", Target: "/mnt", Mode: "bad"},
			},
		},
	}

	if _, err := cfg.runtimeConfig(); err == nil {
		t.Fatalf("expected error for invalid resource set")
	}
}
