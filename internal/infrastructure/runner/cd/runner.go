package cd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

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

type CommandQueryRunner struct{}

func (CommandQueryRunner) Run(ctx context.Context, cwd string, name string, args ...string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("command is required")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
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
