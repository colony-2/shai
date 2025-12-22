//go:build windows

package shai

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/term"
	"golang.org/x/sys/windows"
)

const (
	_WINDOW_BUFFER_SIZE_EVENT = 0x0004
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

	// Do initial resize
	resize()

	// Get the console input handle
	handle := windows.Handle(fd)

	done := make(chan struct{})
	go func() {
		defer func() {
			// Recover from any panics in Windows API calls
			recover()
		}()

		// Buffer for reading console input events
		var inputRecords [1]windows.InputRecord
		var numRead uint32

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-ticker.C:
				// Check if there are any console events available
				var numEvents uint32
				err := windows.GetNumberOfConsoleInputEvents(handle, &numEvents)
				if err != nil || numEvents == 0 {
					continue
				}

				// Read console input events
				err = windows.ReadConsoleInput(handle, &inputRecords[0], 1, &numRead)
				if err != nil || numRead == 0 {
					continue
				}

				// Check if it's a window buffer size event
				event := inputRecords[0]
				if event.EventType == _WINDOW_BUFFER_SIZE_EVENT {
					resize()
				}
			}
		}
	}()

	return func() {
		close(done)
	}
}
