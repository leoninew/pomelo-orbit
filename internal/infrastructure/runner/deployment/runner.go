package deploymentrunner

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

// maxErrorOutputBytes keeps error_message diagnostic without storing full command dumps.
const maxErrorOutputBytes = 4 * 1024

type ShellRunner struct{}

func (ShellRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	commandText := commandDisplayText(name, args...)
	if _, err := fmt.Fprintf(log, "Running: %s\n", commandText); err != nil {
		return err
	}
	var output bytes.Buffer
	writer := io.MultiWriter(log, &output)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Run(); err != nil {
		if _, writeErr := fmt.Fprintf(log, "Command failed: %s: %v\n", commandText, err); writeErr != nil {
			return writeErr
		}
		if detail := commandErrorOutput(output.String()); detail != "" {
			return fmt.Errorf("run %s: %w\n%s", commandText, err, detail)
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

func commandDisplayText(name string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, name)
	parts = append(parts, args...)
	display := make([]string, 0, len(parts))
	for _, part := range parts {
		display = append(display, commandDisplayArg(part))
	}
	return strings.Join(display, " ")
}

func commandDisplayArg(arg string) string {
	if arg == "" || strings.ContainsAny(arg, " \t\r\n\"'") {
		return strconv.Quote(arg)
	}
	return arg
}

func commandErrorOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	if len(output) <= maxErrorOutputBytes {
		return output
	}
	start := len(output) - maxErrorOutputBytes
	if i := strings.IndexByte(output[start:], '\n'); i >= 0 && start+i+1 < len(output) {
		start = start + i + 1
	}
	return "...\n" + output[start:]
}
