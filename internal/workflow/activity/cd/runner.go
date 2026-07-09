package cdsvc

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error
}

type ShellRunner struct{}

func (ShellRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	commandText := shellCommandText(name, args...)
	if _, err := fmt.Fprintf(log, "Working directory: %s\nCommand: %s\n", cwd, commandText); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Run(); err != nil {
		if _, writeErr := fmt.Fprintf(log, "Command failed: %s: %v\n", commandText, err); writeErr != nil {
			return writeErr
		}
		return fmt.Errorf("run %s: %w", commandText, err)
	}
	return nil
}

func shellCommandText(name string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, name)
	parts = append(parts, args...)
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, strconv.Quote(part))
	}
	return strings.Join(quoted, " ")
}
