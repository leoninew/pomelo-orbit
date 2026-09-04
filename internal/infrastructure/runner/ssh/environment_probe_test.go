package sshrunner

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

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
