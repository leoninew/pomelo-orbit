package sshrunner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"golang.org/x/crypto/ssh"
)

const defaultProbeTimeout = 20 * time.Second

type probeError struct {
	diagnostic string
}

func (e probeError) Error() string {
	return e.diagnostic
}

func (e probeError) ProbeDiagnostic() string {
	return e.diagnostic
}

type EnvironmentProber struct {
	dialContext func(context.Context, string, string) (net.Conn, error)
	timeout     time.Duration
}

func NewEnvironmentProber() EnvironmentProber {
	dialer := net.Dialer{}
	return EnvironmentProber{
		dialContext: dialer.DialContext,
		timeout:     defaultProbeTimeout,
	}
}

// Probe opens one strictly pinned SSH session and runs the fixed prerequisite
// command for the configured platform. It does not allocate a PTY, forward
// ports, invoke the system SSH client, or accept arbitrary commands.
func (p EnvironmentProber) Probe(ctx context.Context, environment model.Environment, privateKey credentialdto.DeploymentSSHPrivateKey) (string, error) {
	if !environment.IsSSH() {
		return "", errors.New("SSH environment probe received a non-SSH environment")
	}
	command, err := probeCommand(environment.SSH.Platform)
	if err != nil {
		return "", err
	}
	if p.dialContext == nil {
		return "", errors.New("SSH dialer is not configured")
	}
	if p.timeout <= 0 {
		p.timeout = defaultProbeTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	signer, err := parseSigner(privateKey)
	if err != nil {
		return "", err
	}
	address := net.JoinHostPort(strings.TrimSpace(environment.SSH.Host), strconv.Itoa(environment.SSH.Port))
	connection, err := p.dialContext(ctx, "tcp", address)
	if err != nil {
		return "", probeError{diagnostic: "Cannot connect to the configured SSH host."}
	}
	defer func() { _ = connection.Close() }()
	stopCancelClose := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stopCancelClose()

	var observedFingerprint string
	clientConfig := &ssh.ClientConfig{
		User:            environment.SSH.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: probeHostKeyCallback(environment.SSH.HostKeyFingerprint, &observedFingerprint),
	}
	clientConnection, channels, requests, err := ssh.NewClientConn(connection, address, clientConfig)
	if err != nil {
		if strings.Contains(err.Error(), "SSH host key fingerprint mismatch") {
			return "", probeError{diagnostic: "The SSH host key fingerprint does not match the configured host."}
		}
		return "", probeError{diagnostic: "SSH key authentication failed for the configured user."}
	}
	client := ssh.NewClient(clientConnection, channels, requests)
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return "", probeError{diagnostic: "SSH connected, but a remote session could not be opened."}
	}
	defer func() { _ = session.Close() }()
	session.Stdout = io.Discard
	var stderr bytes.Buffer
	session.Stderr = &stderr
	if err := session.Run(command); err != nil {
		return observedFingerprint, probeError{diagnostic: probePrerequisiteFailureDiagnostic(environment.SSH.Platform, stderr.String())}
	}
	return observedFingerprint, nil
}

func probePrerequisiteFailureDiagnostic(platform string, stderr string) string {
	if platform != model.EnvironmentPlatformWindows {
		return "Docker and Docker Compose prerequisites are not ready on the SSH target."
	}
	switch {
	case strings.Contains(stderr, "WSL2 Docker Desktop distribution is unavailable"):
		return "WSL2 and Docker Desktop must be installed and running."
	case strings.Contains(stderr, "Docker Desktop engine is unavailable"):
		return "Docker Desktop is running, but its engine is unavailable."
	case strings.Contains(stderr, "Docker Compose is unavailable"):
		return "Docker Compose is unavailable."
	case strings.Contains(stderr, "Docker Desktop must use Linux containers"):
		return "Docker Desktop must use Linux containers."
	case strings.Contains(stderr, "Docker daemon is unavailable"):
		return "Docker Desktop is running, but its Docker daemon is unavailable."
	default:
		return "Windows Docker prerequisites are not ready on the SSH target."
	}
}

func parseSigner(privateKey credentialdto.DeploymentSSHPrivateKey) (ssh.Signer, error) {
	if privateKey.Passphrase == "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey.PrivateKey))
		if err != nil {
			return nil, errors.New("parse deployment SSH private key")
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey.PrivateKey), []byte(privateKey.Passphrase))
	if err != nil {
		return nil, errors.New("parse deployment SSH private key")
	}
	return signer, nil
}

func strictHostKeyCallback(expected string) ssh.HostKeyCallback {
	expected = strings.TrimSpace(expected)
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if expected == "" || ssh.FingerprintSHA256(key) != expected {
			return errors.New("SSH host key fingerprint mismatch")
		}
		return nil
	}
}

func probeHostKeyCallback(expected string, observed *string) ssh.HostKeyCallback {
	expected = strings.TrimSpace(expected)
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		fingerprint := ssh.FingerprintSHA256(key)
		if observed != nil {
			*observed = fingerprint
		}
		if expected != "" && fingerprint != expected {
			return errors.New("SSH host key fingerprint mismatch")
		}
		return nil
	}
}

func probeCommand(platform string) (string, error) {
	switch platform {
	case model.EnvironmentPlatformLinux:
		return `sh -lc 'set -eu; command -v docker >/dev/null; docker version --format "{{.Server.Version}}" >/dev/null; docker compose version >/dev/null; docker info >/dev/null'`, nil
	case model.EnvironmentPlatformWindows:
		script := `$ErrorActionPreference = 'Stop'; $distros = (& wsl.exe -l -v | Out-String) -replace [char]0, ''; if ($LASTEXITCODE -ne 0 -or $distros -notmatch '(?m)^\s*\*?\s*docker-desktop\s+\S+\s+2\s*$') { throw 'WSL2 Docker Desktop distribution is unavailable' }; & docker.exe version --format '{{.Server.Version}}' | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'Docker Desktop engine is unavailable' }; & docker.exe compose version | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'Docker Compose is unavailable' }; $serverOS = (& docker.exe version --format '{{.Server.Os}}').Trim(); if ($LASTEXITCODE -ne 0 -or $serverOS -ne 'linux') { throw 'Docker Desktop must use Linux containers' }; & docker.exe info | Out-Null; if ($LASTEXITCODE -ne 0) { throw 'Docker daemon is unavailable' }`
		return powerShellEncodedCommand(script), nil
	default:
		return "", errors.New("unsupported environment platform")
	}
}
