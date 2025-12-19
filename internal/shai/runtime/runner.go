package shai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/colony-2/shai/internal/container"
	configpkg "github.com/colony-2/shai/internal/shai/runtime/config"
)

// Config represents shai configuration
type Config struct {
	WorkingDir     string // Workspace root containing .shai/config.yaml
	ConfigFile     string // Path to .shai/config.yaml (optional)
	TemplateVars   map[string]string
	ReadWritePaths []string // Paths to mount as read-write
	ContainerName  string   // Optional container name
}

// Runner coordinates shai operations
type Runner struct {
	config           Config
	shaiConfig       *configpkg.Config
	manager          container.Manager
	mountBuilder     *MountBuilder
	progressReporter *ProgressReporter
}

// New creates a new shai runner with a provided container manager
func New(config Config, manager container.Manager) (*Runner, error) {
	// Use current directory if not specified
	if config.WorkingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		config.WorkingDir = wd
	}

	configPath := config.ConfigFile
	if configPath == "" {
		configPath = filepath.Join(config.WorkingDir, DefaultConfigRelPath)
	}
	shaiCfg, _, err := configpkg.LoadOrDefault(configPath, hostEnvMap(), config.TemplateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to load shai config: %w", err)
	}

	// Create mount builder
	mountBuilder, err := NewMountBuilder(config.WorkingDir, config.ReadWritePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to create mount builder: %w", err)
	}

	return &Runner{
		config:           config,
		shaiConfig:       shaiCfg,
		manager:          manager,
		mountBuilder:     mountBuilder,
		progressReporter: NewProgressReporter(),
	}, nil
}

// OnProgress sets the progress callback
func (r *Runner) OnProgress(cb ProgressCallback) {
	r.progressReporter.SetCallback(cb)
}

// Start creates and starts the container
func (r *Runner) Start(ctx context.Context) (*container.Info, error) {
	r.progressReporter.Report(PhaseValidating, "Configuring selective mounts...")

	// Configure custom mounts on the manager
	if err := r.configureManagerMounts(); err != nil {
		return nil, fmt.Errorf("failed to configure mounts: %w", err)
	}

	r.progressReporter.Report(PhaseCreating, "Creating container with custom mounts...")

	containerID, err := r.manager.Create(ctx, r.config.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	r.progressReporter.Report(PhaseStarting, "Starting container...")

	// Start container
	err = r.manager.Start(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	r.progressReporter.Report(PhaseStarting, "Starting interactive shell")

	// Get container info
	info, err := r.manager.GetInfo(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	return info, nil
}

// AttachInteractive attaches an interactive shell
func (r *Runner) AttachInteractive(ctx context.Context, containerID string) error {
	// Check if the manager supports interactive attachment
	if attachable, ok := r.manager.(interface {
		AttachInteractive(context.Context, string) error
	}); ok {
		return attachable.AttachInteractive(ctx, containerID)
	}
	return fmt.Errorf("manager does not support interactive attachment")
}

// Close cleans up resources
func (r *Runner) Close() error {
	// Check if the manager implements Close
	if closer, ok := r.manager.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// configureManagerMounts configures the container manager with our custom mounts
func (r *Runner) configureManagerMounts() error {
	// Build mount configurations from our mount builder
	dockerMounts := r.mountBuilder.BuildMounts()

	// Convert Docker mount.Mount to container.Mount
	var containerMounts []container.Mount
	for _, m := range dockerMounts {
		containerMounts = append(containerMounts, container.Mount{
			Type:     string(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// Configure the manager with our custom mounts
	return r.manager.ConfigureMounts(containerMounts)
}
