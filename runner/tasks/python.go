package tasks

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	pythonTimeout   = 5 * time.Minute
	pythonMaxLine   = 1 << 20
	pythonWaitDelay = 5 * time.Second

	// OutputPrefix marks program output lines; the backend never treats them as terminal,
	// so a script printing "ERROR: ..." or "... completed" cannot finish the job early.
	OutputPrefix = "| "
)

// RunPython executes in.Code with python3, streaming combined stdout/stderr as log lines.
// The script runs unsandboxed with the runner's permissions.
func RunPython(ctx context.Context, in Input, logf Logf) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(in.Code) == "" {
		return errors.New("no Python code provided")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		return fmt.Errorf("python3 not found: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, pythonTimeout)
	defer cancel()

	// "-" reads the script from stdin; -u disables buffering so output streams live.
	cmd := exec.CommandContext(runCtx, python, "-u", "-")
	cmd.Stdin = strings.NewReader(in.Code)
	cmd.WaitDelay = pythonWaitDelay
	// Kill the whole process group: python3 may be a launcher shim (e.g. macOS /usr/bin/python3),
	// and the script can spawn children, any of which would otherwise keep the output pipe open.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	logf("Running Python script with %s...", python)
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 0, 64*1024), pythonMaxLine)
	for scanner.Scan() {
		logf("%s%s", OutputPrefix, scanner.Text())
	}
	scanErr := scanner.Err()
	if scanErr != nil {
		cancel()
	}
	waitErr := cmd.Wait()

	switch {
	case ctx.Err() != nil:
		return ctx.Err()
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		return fmt.Errorf("python script timed out after %s", pythonTimeout)
	case scanErr != nil:
		return fmt.Errorf("reading python output: %w", scanErr)
	case waitErr != nil:
		return fmt.Errorf("python script failed: %w", waitErr)
	}
	logf("Python script completed")
	return nil
}
