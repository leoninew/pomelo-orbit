package sshrunner

import (
	"context"
	"errors"
	"fmt"
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
func (p EnvironmentProber) Probe(ctx context.Context, environment model.Environment, privateKey credentialdto.DeploymentSSHPrivateKey) error {
	command, err := probeCommand(environment.Platform)
	if err != nil {
		return err
	}
	if p.dialContext == nil {
		return errors.New("SSH dialer is not configured")
	}
	if p.timeout <= 0 {
		p.timeout = defaultProbeTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	signer, err := parseSigner(privateKey)
	if err != nil {
		return err
	}
	address := net.JoinHostPort(strings.TrimSpace(environment.Host), strconv.Itoa(environment.Port))
	connection, err := p.dialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial SSH target: %w", err)
	}
	defer func() { _ = connection.Close() }()
	stopCancelClose := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stopCancelClose()

	clientConfig := &ssh.ClientConfig{
		User:            environment.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: strictHostKeyCallback(environment.HostKeyFingerprint),
	}
	clientConnection, channels, requests, err := ssh.NewClientConn(connection, address, clientConfig)
	if err != nil {
		return fmt.Errorf("establish SSH connection: %w", err)
	}
	client := ssh.NewClient(clientConnection, channels, requests)
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create SSH session: %w", err)
	}
	defer func() { _ = session.Close() }()
	session.Stdout = io.Discard
	session.Stderr = io.Discard
	if err := session.Run(command); err != nil {
		return fmt.Errorf("run SSH environment probe: %w", err)
	}
	return nil
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
