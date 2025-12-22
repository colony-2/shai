//go:build windows

package shai

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/term"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procReadConsoleInput           = kernel32.NewProc("ReadConsoleInputW")
	procGetNumberOfConsoleInputEvents = kernel32.NewProc("GetNumberOfConsoleInputEvents")
)

const (
	_WINDOW_BUFFER_SIZE_EVENT = 0x0004
)

// INPUT_RECORD structure from Windows Console API
type inputRecord struct {
	EventType uint16
	_         uint16 // padding
	Event     [16]byte
}

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

	done := make(chan struct{})
	go func() {
		defer func() {
			// Recover from any panics in Windows API calls
			recover()
		}()

		handle := syscall.Handle(fd)
		var inputRecords [128]inputRecord
		var numRead uint32

		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			default:
			}

			// Check if there are console input events
			var numEvents uint32
			ret, _, _ := procGetNumberOfConsoleInputEvents.Call(
				uintptr(handle),
				uintptr(unsafe.Pointer(&numEvents)),
			)
			if ret == 0 || numEvents == 0 {
				// Sleep briefly to avoid busy loop
				select {
				case <-ctx.Done():
					return
				case <-done:
					return
				case <-func() <-chan struct{} {
					ch := make(chan struct{})
					go func() {
						syscall.Sleep(50) // 50ms
						close(ch)
					}()
					return ch
				}():
				}
				continue
			}

			// Read console input events
			ret, _, _ = procReadConsoleInput.Call(
				uintptr(handle),
				uintptr(unsafe.Pointer(&inputRecords[0])),
				uintptr(len(inputRecords)),
				uintptr(unsafe.Pointer(&numRead)),
			)
			if ret == 0 || numRead == 0 {
				continue
			}

			// Check for window buffer size events
			for i := uint32(0); i < numRead; i++ {
				if inputRecords[i].EventType == _WINDOW_BUFFER_SIZE_EVENT {
					resize()
					break
				}
			}
		}
	}()

	return func() {
		close(done)
	}
}
