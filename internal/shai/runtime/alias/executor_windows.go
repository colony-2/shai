//go:build windows

package alias

import (
	"os/exec"
	"time"
)

func configureProcAttr(cmd *exec.Cmd) {
	// On Windows, we don't need to set process group attributes
	// Process management works differently than Unix
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// On Windows, we can't use process groups like Unix
		// Just kill the process directly
		_ = cmd.Process.Kill()
		time.Sleep(250 * time.Millisecond)
	}
}
