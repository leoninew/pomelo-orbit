package sshrunner

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"golang.org/x/crypto/ssh"
)

func TestStrictHostKeyCallbackAcceptsOnlyConfiguredFingerprint(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	callback := strictHostKeyCallback(ssh.FingerprintSHA256(signer.PublicKey()))
	if err := callback("example.test:22", nil, signer.PublicKey()); err != nil {
		t.Fatalf("matching fingerprint rejected: %v", err)
	}
	if err := strictHostKeyCallback("SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")("example.test:22", nil, signer.PublicKey()); err == nil {
		t.Fatal("unexpected host key acceptance")
	}
}

func TestProbeHostKeyCallbackAcceptsEmptyExpectedAndRecordsFingerprint(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	var observed string
	if err := probeHostKeyCallback("", &observed)("example.test:22", nil, signer.PublicKey()); err != nil {
		t.Fatalf("empty expected fingerprint rejected: %v", err)
	}
	if observed != ssh.FingerprintSHA256(signer.PublicKey()) {
		t.Fatalf("observed fingerprint = %q", observed)
	}
	if err := probeHostKeyCallback("SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", &observed)("example.test:22", nil, signer.PublicKey()); err == nil {
		t.Fatal("mismatched expected fingerprint accepted")
	}
}

func TestTestSSHReportsUnreachableHost(t *testing.T) {
	prober := EnvironmentProber{
		dialContext: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("connection refused")
		},
		timeout: time.Second,
	}
	err := prober.TestSSH(context.Background(), "192.0.2.10", 22, "orbit")
	if err == nil || !strings.Contains(err.Error(), "Cannot connect to the configured SSH host") {
		t.Fatalf("error = %v", err)
	}
}

func TestSSHServiceReachableTreatsAuthenticationFailureAsReachable(t *testing.T) {
	if !sshServiceReachable(errors.New("ssh: handshake failed: ssh: unable to authenticate, no supported methods remain")) {
		t.Fatal("authentication failure must mean the SSH service is reachable")
	}
	if sshServiceReachable(errors.New("connection refused")) {
		t.Fatal("connection refused must not mean the SSH service is reachable")
	}
}

func TestProbeCommandIsFixedPerSupportedPlatform(t *testing.T) {
	linux, err := probeCommand(model.EnvironmentPlatformLinux)
	if err != nil || !strings.HasPrefix(linux, "sh -lc ") || !strings.Contains(linux, "docker compose version") {
		t.Fatalf("linux command = %q, %v", linux, err)
	}
	windows, err := probeCommand(model.EnvironmentPlatformWindows)
	if err != nil {
		t.Fatalf("windows probe command = %q, %v", windows, err)
	}
	windowsScript := decodePowerShellEncodedCommand(t, windows)
	if !strings.Contains(windowsScript, "wsl.exe -l -v") || !strings.Contains(windowsScript, "-replace [char]0, ''") ||
		!strings.Contains(windowsScript, "docker.exe compose version") || !strings.Contains(windowsScript, "{{.Server.Os}}") ||
		strings.Contains(windowsScript, "wsl.exe -d docker-desktop") {
		t.Fatalf("windows probe script = %q", windowsScript)
	}
	if _, err := probeCommand("darwin"); err == nil {
		t.Fatal("unsupported platform command unexpectedly succeeded")
	}
}
