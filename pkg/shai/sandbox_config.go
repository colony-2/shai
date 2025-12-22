package shai

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	runtimepkg "github.com/colony-2/shai/internal/shai/runtime"
)

// SandboxConfig describes how to launch a sandbox.
type SandboxConfig struct {
	WorkingDir          string
	ConfigFile          string
	TemplateVars        map[string]string
	ReadWritePaths      []string
	ResourceSets        []string
	Verbose             bool
	PostSetupExec       *SandboxExec
	Stdout              io.Writer
	Stderr              io.Writer
	GracefulStopTimeout time.Duration
	ImageOverride       string
	UserOverride        string
	HostUID             string
	HostGID             string
	Privileged          bool
	ShowProgress        bool
}

// SandboxExec describes a command to run inside the sandbox after setup.
type SandboxExec struct {
	Command []string
	Env     map[string]string
	Workdir string
	UseTTY  bool
}

// SandboxConfigOption mutates a SandboxConfig during construction.
type SandboxConfigOption func(*SandboxConfig)

// LoadSandboxConfig initializes a SandboxConfig rooted at workspace and applies optional overrides.
func LoadSandboxConfig(workspace string, opts ...SandboxConfigOption) (SandboxConfig, error) {
	cfg := SandboxConfig{
		WorkingDir: workspace,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if err := cfg.normalize(); err != nil {
		return SandboxConfig{}, err
	}
	return cfg, nil
}

// WithConfigFile overrides the default .shai config path.
func WithConfigFile(path string) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.ConfigFile = path
	}
}

// WithTemplateVars sets template variables for config evaluation.
func WithTemplateVars(vars map[string]string) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.TemplateVars = vars
	}
}

// WithResourceSets preselects resource sets.
func WithResourceSets(names []string) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.ResourceSets = names
	}
}

// WithReadWritePaths sets read-write mount points.
func WithReadWritePaths(paths []string) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.ReadWritePaths = paths
	}
}

// WithStdout directs non-TTY stdout to writer.
func WithStdout(w io.Writer) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.Stdout = w
	}
}

// WithStderr directs non-TTY stderr to writer.
func WithStderr(w io.Writer) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.Stderr = w
	}
}

// WithImageOverride forces a container image.
func WithImageOverride(image string) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.ImageOverride = image
	}
}

// WithUserOverride forces a target user.
func WithUserOverride(user string) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.UserOverride = user
	}
}

// WithVerbose toggles verbose logging.
func WithVerbose(verbose bool) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.Verbose = verbose
	}
}

// WithGracefulStopTimeout overrides the shutdown grace period.
func WithGracefulStopTimeout(d time.Duration) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.GracefulStopTimeout = d
	}
}

func (cfg SandboxConfig) runtimeConfig() runtimepkg.EphemeralConfig {
	normalized := cfg
	_ = normalized.normalize()
	return runtimepkg.EphemeralConfig{
		WorkingDir:          normalized.WorkingDir,
		ConfigFile:          normalized.ConfigFile,
		TemplateVars:        normalized.TemplateVars,
		ReadWritePaths:      normalized.ReadWritePaths,
		ResourceSets:        normalized.ResourceSets,
		Verbose:             normalized.Verbose,
		PostSetupExec:       convertExec(normalized.PostSetupExec),
		Stdout:              normalized.Stdout,
		Stderr:              normalized.Stderr,
		GracefulStopTimeout: normalized.GracefulStopTimeout,
		ImageOverride:       normalized.ImageOverride,
		UserOverride:        normalized.UserOverride,
		HostUID:             normalized.HostUID,
		HostGID:             normalized.HostGID,
		Privileged:          normalized.Privileged,
		ShowProgress:        normalized.ShowProgress,
	}
}

func convertExec(exec *SandboxExec) *runtimepkg.ExecSpec {
	if exec == nil {
		return nil
	}
	return &runtimepkg.ExecSpec{
		Command: exec.Command,
		Env:     exec.Env,
		Workdir: exec.Workdir,
		UseTTY:  exec.UseTTY,
	}
}

func (cfg *SandboxConfig) normalize() error {
	if cfg == nil {
		return nil
	}
	if strings.TrimSpace(cfg.WorkingDir) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		cfg.WorkingDir = wd
	}
	if strings.TrimSpace(cfg.ConfigFile) == "" {
		cfg.ConfigFile = filepath.Join(cfg.WorkingDir, DefaultConfigRelPath)
	}
	return nil
}
