package sshrunner

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

// Component tests exercise real SSH channels, EOF, exit-status and cancellation.
func TestCommandSSHStreamsEOFAndExitStatus(t *testing.T) {
	target, server := startSessionServer(t)
	session, err := NewCommandRunner().Open(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	if len(server.commands) != 0 || session.HostKeyFingerprint() != target.Environment.SSH.HostKeyFingerprint {
		t.Fatal("Open sent a command or lost host key metadata")
	}
	var out, errOut bytes.Buffer
	input := []byte{0, 255, 'a', '\n'}
	result, err := session.Run(context.Background(), environmentport.Command{Argv: []string{"cli-io", "$HOME", "a b"}, Workdir: "~", Stdin: input}, &out, &errOut)
	if err != nil || !result.Sent || result.ExitCode == nil || *result.ExitCode != 7 || !reflect.DeepEqual(out.Bytes(), input) || errOut.String() != "stderr-data" {
		t.Fatalf("result=%+v out=%q stderr=%q err=%v", result, out.Bytes(), errOut.String(), err)
	}
}

func TestCommandSSHLostStatusAndCancellationAreNotSuccess(t *testing.T) {
	for _, program := range []string{"missing-status", "block"} {
		t.Run(program, func(t *testing.T) {
			target, server := startSessionServer(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			session, err := NewCommandRunner().Open(ctx, target)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = session.Close() }()
			done := make(chan struct{})
			var result environmentport.CommandOutcome
			go func() {
				result, err = session.Run(ctx, environmentport.Command{Argv: []string{program}, Workdir: "~"}, &bytes.Buffer{}, &bytes.Buffer{})
				close(done)
			}()
			if program == "block" {
				select {
				case <-server.blocked:
					cancel()
				case <-ctx.Done():
					t.Fatal("exec did not reach server")
				}
			}
			select {
			case <-done:
			case <-ctx.Done():
				if program != "block" {
					t.Fatal("missing status hung")
				}
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("cancel did not close SSH connection")
				}
			}
			if !result.Sent || result.ExitCode != nil || err == nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestGenericCommandPreservesLiteralArguments(t *testing.T) {
	command, err := genericCommand("linux", "~/work", []string{"curl", "~", "$HOME", "*", "a'b", ""})
	if err != nil {
		t.Fatal(err)
	}
	expected := `cd "$HOME"'/work' && exec 'curl' '~' '$HOME' '*' 'a'"'"'b' ''`
	if command != expected {
		t.Fatalf("command=%s", command)
	}
	command, err = genericCommand("windows", `C:\work`, []string{"curl", "~", "$HOME", ""})
	if err != nil || !strings.HasPrefix(command, powerShellEncodedCommandPrefix) {
		t.Fatalf("command=%q err=%v", command, err)
	}
}
