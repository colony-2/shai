package alias

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Streams configures stdout/stderr destinations for alias execution.
type Streams struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Executor runs alias commands on the host.
type Executor struct {
	WorkingDir string
	ShellPath  string
	Timeout    time.Duration
}

// RunResult captures the outcome of an alias command.
type RunResult struct {
	ExitCode int
}

// Run executes the provided alias entry using the configured shell.
func (e *Executor) Run(ctx context.Context, entry *Entry, args []string, streams Streams) (RunResult, error) {
	if entry == nil {
		return RunResult{}, fmt.Errorf("alias entry is nil")
	}
	argString := strings.TrimSpace(strings.Join(args, " "))
	if err := entry.ValidateArgs(argString); err != nil {
		return RunResult{}, err
	}

	commandLine := entry.Command
	if argString != "" {
		commandLine = commandLine + " " + argString
	}

	shell := e.ShellPath
	if strings.TrimSpace(shell) == "" {
		shell = "/bin/bash"
	}
	if _, err := os.Stat(shell); err != nil {
		return RunResult{}, fmt.Errorf("shell %q unavailable: %w", shell, err)
	}

	timeout := e.Timeout
	execCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, shell, "-lc", commandLine)
	cmd.Dir = e.WorkingDir
	cmd.Env = os.Environ()
	configureProcAttr(cmd)
	cmd.Stdout = writerOrDiscard(streams.Stdout)
	cmd.Stderr = writerOrDiscard(streams.Stderr)

	exited := make(chan struct{})
	go func() {
		<-execCtx.Done()
		select {
		case <-exited:
			return
		default:
		}
		killProcessGroup(cmd)
	}()

	if err := cmd.Start(); err != nil {
		close(exited)
		return RunResult{}, err
	}
	waitErr := cmd.Wait()
	close(exited)

	if waitErr == nil {
		return RunResult{ExitCode: 0}, nil
	}
	if execCtx.Err() != nil && errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		return RunResult{}, fmt.Errorf("alias command timed out")
	}
	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		return RunResult{ExitCode: exitErr.ExitCode()}, nil
	}
	return RunResult{}, waitErr
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return io.Discard
}
