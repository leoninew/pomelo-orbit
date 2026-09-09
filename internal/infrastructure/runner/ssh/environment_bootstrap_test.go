package sshrunner

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os/exec"
	"strings"
	"testing"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"golang.org/x/crypto/ssh"
)

func TestBootstrapAuthMethodsAcceptPasswordOrPrivateKey(t *testing.T) {
	passwordMethods, err := bootstrapAuthMethods(environmentport.BootstrapAuth{Username: "opc", Password: "secret"})
	if err != nil || len(passwordMethods) != 1 {
		t.Fatalf("password auth methods = %#v, %v", passwordMethods, err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "bootstrap-test")
	if err != nil {
		t.Fatal(err)
	}
	privateKeyMethods, err := bootstrapAuthMethods(environmentport.BootstrapAuth{
		Username: "opc", PrivateKey: string(pem.EncodeToMemory(block)),
	})
	if err != nil || len(privateKeyMethods) != 1 {
		t.Fatalf("private key auth methods = %#v, %v", privateKeyMethods, err)
	}

	if _, err := bootstrapAuthMethods(environmentport.BootstrapAuth{Username: "opc"}); err == nil {
		t.Fatal("missing authentication unexpectedly accepted")
	}
	if _, err := bootstrapAuthMethods(environmentport.BootstrapAuth{Username: "opc", Password: "secret", PrivateKey: "key"}); err == nil {
		t.Fatal("ambiguous authentication unexpectedly accepted")
	}
}

func TestBootstrapUsesTOFUOrPinnedHostKeyCallback(t *testing.T) {
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
		t.Fatalf("TOFU callback rejected host key: %v", err)
	}
	if observed != ssh.FingerprintSHA256(signer.PublicKey()) {
		t.Fatalf("observed fingerprint = %q", observed)
	}
	if err := probeHostKeyCallback("SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", nil)("example.test:22", nil, signer.PublicKey()); err == nil {
		t.Fatal("mismatched pinned host key unexpectedly accepted")
	}
}

func TestLinuxEnvironmentBootstrapCommandChecksPrerequisitesBeforeWriting(t *testing.T) {
	command, err := linuxEnvironmentBootstrapCommand(model.EnvironmentSSHTarget{
		Platform: model.EnvironmentPlatformLinux, Username: "deploy", WorkspaceRoot: "~/.pomelo-orbit",
	}, "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample comment")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"sudo -n -v",
		"docker compose version >/dev/null",
		"sshd -t",
		"pubkeyauthentication",
		"authorized_keys",
		"sudo -n install -d -m 0750 -o 'deploy'",
	} {
		if !strings.Contains(command, expected) {
			t.Fatalf("bootstrap command is missing %q: %q", expected, command)
		}
	}
	for _, forbidden := range []string{
		"download.docker.com",
		"docker-ce",
		"usermod -aG docker",
		"systemctl restart docker",
		"firewall-cmd",
		"apt-get install",
		"systemctl restart sshd",
	} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("bootstrap command must not mutate unrelated host state: found %q", forbidden)
		}
	}
	if strings.Index(command, "docker compose version") > strings.Index(command, "authorized_keys") {
		t.Fatal("Docker prerequisites must be checked before writing the deployment key")
	}

	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell is unavailable")
	}
	process := exec.Command(shell, "-n")
	process.Stdin = strings.NewReader(command)
	if output, err := process.CombinedOutput(); err != nil {
		t.Fatalf("Linux bootstrap command has a POSIX syntax error: %v: %s", err, output)
	}
}

func TestBootstrapDiagnosticsAreBoundedAndDoNotExposeRemoteOutput(t *testing.T) {
	secret := "bootstrap-private-key-must-not-appear"
	output := strings.Repeat("x", maxBootstrapErrorOutputBytes+1024) + secret
	var limited limitedBootstrapOutput
	if written, err := limited.Write([]byte(output)); err != nil || written != len(output) {
		t.Fatalf("limited write = %d, %v", written, err)
	}
	if len(limited.String()) != maxBootstrapErrorOutputBytes {
		t.Fatalf("limited output length = %d", len(limited.String()))
	}
	if diagnostic := bootstrapFailureDiagnostic(limited.String()); strings.Contains(diagnostic, secret) || diagnostic == "" {
		t.Fatalf("unsafe bootstrap diagnostic = %q", diagnostic)
	}
}
