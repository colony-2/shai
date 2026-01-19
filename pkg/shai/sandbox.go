package shai

import (
	"context"

	runtimepkg "github.com/colony-2/shai/internal/shai/runtime"
)

// Sandbox provides lifecycle operations for an ephemeral Shai environment.
type Sandbox interface {
	Run(ctx context.Context) error
	Start(ctx context.Context) (*SandboxSession, error)
	Close() error
}

// SandboxSession supervises a non-blocking sandbox execution.
type SandboxSession struct {
	ContainerID string

	session *runtimepkg.Session
}

// Wait blocks until the sandbox stops or ctx is cancelled.
func (s *SandboxSession) Wait(ctx context.Context) error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Wait(ctx)
}

// Stop requests a graceful shutdown.
func (s *SandboxSession) Stop(ctx context.Context) error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Stop(ctx)
}

// Close releases session resources.
func (s *SandboxSession) Close() error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Close()
}

// NewSandbox creates a new sandbox using the provided configuration.
func NewSandbox(cfg SandboxConfig) (Sandbox, error) {
	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	rc, err := cfg.runtimeConfig()
	if err != nil {
		return nil, err
	}
	runner, err := runtimepkg.NewEphemeralRunner(rc)
	if err != nil {
		return nil, err
	}
	return &sandboxImpl{runner: runner}, nil
}

type sandboxImpl struct {
	runner *runtimepkg.EphemeralRunner
}

func (s *sandboxImpl) Run(ctx context.Context) error {
	return s.runner.Run(ctx)
}

func (s *sandboxImpl) Start(ctx context.Context) (*SandboxSession, error) {
	session, err := s.runner.Start(ctx)
	if err != nil {
		return nil, err
	}
	return &SandboxSession{ContainerID: session.ContainerID, session: session}, nil
}

func (s *sandboxImpl) Close() error {
	return s.runner.Close()
}
