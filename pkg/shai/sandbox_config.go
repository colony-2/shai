package shai

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	runtimepkg "github.com/colony-2/shai/internal/shai/runtime"
	configpkg "github.com/colony-2/shai/internal/shai/runtime/config"
)

// SandboxConfig describes how to launch a sandbox.
type SandboxConfig struct {
	WorkingDir          string
	ConfigFile          string
	TemplateVars        map[string]string
	ReadWritePaths      []string
	ResourceSets        []string
	PrependResourceSet  *ResourceSet
	AppendResourceSet   *ResourceSet
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
	ShowScriptOutput    bool
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

// WithPrependResourceSet prepends a resource set before resolved resources.
func WithPrependResourceSet(set *ResourceSet) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.PrependResourceSet = set
	}
}

// WithAppendResourceSet appends a resource set after resolved resources.
func WithAppendResourceSet(set *ResourceSet) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.AppendResourceSet = set
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

// WithShowScriptOutput toggles bootstrap script output (warnings/banner).
func WithShowScriptOutput(enabled bool) SandboxConfigOption {
	return func(cfg *SandboxConfig) {
		cfg.ShowScriptOutput = enabled
	}
}

func (cfg SandboxConfig) runtimeConfig() (runtimepkg.EphemeralConfig, error) {
	normalized := cfg
	_ = normalized.normalize()
	prepend, err := convertResourceSet(normalized.PrependResourceSet, "prepend")
	if err != nil {
		return runtimepkg.EphemeralConfig{}, err
	}
	appendSet, err := convertResourceSet(normalized.AppendResourceSet, "append")
	if err != nil {
		return runtimepkg.EphemeralConfig{}, err
	}
	return runtimepkg.EphemeralConfig{
		WorkingDir:          normalized.WorkingDir,
		ConfigFile:          normalized.ConfigFile,
		TemplateVars:        normalized.TemplateVars,
		ReadWritePaths:      normalized.ReadWritePaths,
		ResourceSets:        normalized.ResourceSets,
		PrependResourceSet:  prepend,
		AppendResourceSet:   appendSet,
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
		ShowScriptOutput:    normalized.ShowScriptOutput,
	}, nil
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

func convertResourceSet(set *ResourceSet, label string) (*configpkg.ResourceSet, error) {
	if set == nil {
		return nil, nil
	}
	out := &configpkg.ResourceSet{
		Vars:         make([]configpkg.VarMapping, len(set.Vars)),
		Mounts:       make([]configpkg.Mount, len(set.Mounts)),
		Calls:        make([]configpkg.Call, len(set.Calls)),
		HTTP:         append([]string{}, set.HTTP...),
		Ports:        make([]configpkg.Port, len(set.Ports)),
		RootCommands: append([]string{}, set.RootCommands...),
		Options: configpkg.ResourceOptions{
			Privileged: set.Options.Privileged,
		},
	}
	for i, vm := range set.Vars {
		out.Vars[i] = configpkg.VarMapping{Source: vm.Source, Target: vm.Target}
	}
	for i, m := range set.Mounts {
		out.Mounts[i] = configpkg.Mount{Source: m.Source, Target: m.Target, Mode: m.Mode}
	}
	for i, c := range set.Calls {
		out.Calls[i] = configpkg.Call{
			Name:        c.Name,
			Description: c.Description,
			Command:     c.Command,
			AllowedArgs: c.AllowedArgs,
		}
	}
	for i, p := range set.Ports {
		out.Ports[i] = configpkg.Port{Host: p.Host, Port: p.Port}
	}
	if err := configpkg.NormalizeResourceSet(out, label); err != nil {
		return nil, err
	}
	return out, nil
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
