//go:build !windows

package shai

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/term"
)

func (r *EphemeralRunner) startTTYResizeWatcher(ctx context.Context, fd uintptr, containerID string) func() {
	if !term.IsTerminal(fd) {
		return nil
	}
	resize := func() {
		if ws, err := term.GetWinsize(fd); err == nil && ws != nil {
			_ = r.docker.ContainerResize(context.Background(), containerID, container.ResizeOptions{
				Height: uint(ws.Height),
				Width:  uint(ws.Width),
			})
		}
	}
	resize()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)

	done := make(chan struct{})
	go func() {
		defer signal.Stop(sigCh)
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-sigCh:
				resize()
			}
		}
	}()

	return func() {
		close(done)
	}
}
