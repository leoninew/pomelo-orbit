package sshrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Local process component coverage of Windows PowerShell native argument and
// byte-stream forwarding. This doesn't replace a real OpenSSH smoke test.
func TestWindowsCommandPreservesNativeArgvAndByteStreams(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"", "a b", `a"b`, `trailing\`, `$HOME`, `~`, `[literal]`, "中"}
	command, err := genericCommand("windows", t.TempDir(), append([]string{executable, "-test.run=TestCommandArgvHelperProcess", "--"}, args...))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	process := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-OutputFormat", "Text", "-EncodedCommand", strings.TrimPrefix(command, powerShellEncodedCommandPrefix))
	process.Env = append(os.Environ(), "POMELO_ORBIT_TEST_COMMAND_HELPER=1")
	input := []byte{0, 255, 'a', '\n'}
	process.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	process.Stdout = &stdout
	process.Stderr = &stderr
	err = process.Run()
	if process.ProcessState == nil || process.ProcessState.ExitCode() != 7 {
		t.Fatalf("native exit err=%v stderr=%q", err, stderr.String())
	}
	header, body, ok := bytes.Cut(stdout.Bytes(), []byte{'\n'})
	var actual []string
	if !ok || json.Unmarshal(header, &actual) != nil || !reflect.DeepEqual(actual, args) || !bytes.Equal(body, input) || !bytes.Equal(stderr.Bytes(), []byte{0, 255}) {
		t.Fatalf("argv=%q stdout=%q stderr=%q", actual, stdout.Bytes(), stderr.Bytes())
	}
}

func TestCommandArgvHelperProcess(t *testing.T) {
	if os.Getenv("POMELO_ORBIT_TEST_COMMAND_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			data, _ := json.Marshal(os.Args[i+1:])
			_, _ = os.Stdout.Write(append(data, '\n'))
			break
		}
	}
	_, _ = io.Copy(os.Stdout, os.Stdin)
	_, _ = os.Stderr.Write([]byte{0, 255})
	os.Exit(7)
}
