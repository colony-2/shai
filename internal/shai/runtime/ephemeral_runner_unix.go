//go:build !windows

package shai

import (
	"context"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sort"
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

func dockerSocketCandidates() []string {
	seen := make(map[string]bool)
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
	}
	add("/var/run/docker.sock")
	add("/run/docker.sock")
	add("/var/run/podman/podman.sock")
	add("/run/podman/podman.sock")

	if home := os.Getenv("HOME"); home != "" {
		add(filepath.Join(home, ".docker", "run", "docker.sock"))
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		add(filepath.Join(xdg, "docker.sock"))
		add(filepath.Join(xdg, "podman", "podman.sock"))
	}
	if current, err := user.Current(); err == nil && current.Uid != "" {
		add(filepath.Join("/run/user", current.Uid, "docker.sock"))
		add(filepath.Join("/run/user", current.Uid, "podman/podman.sock"))
	} else if uid := os.Getenv("UID"); uid != "" {
		add(filepath.Join("/run/user", uid, "docker.sock"))
		add(filepath.Join("/run/user", uid, "podman/podman.sock"))
	}

	paths := make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
