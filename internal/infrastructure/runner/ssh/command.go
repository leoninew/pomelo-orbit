package sshrunner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"golang.org/x/crypto/ssh"
)

type CommandRunner struct{ runtime *Runtime }

func NewCommandRunner() *CommandRunner { return &CommandRunner{runtime: NewRuntime()} }

func (r *CommandRunner) Open(ctx context.Context, target environmentport.Target) (environmentport.CommandSession, error) {
	var fingerprint string
	client, err := r.runtime.dialSSHWithHostKey(ctx, target, &fingerprint)
	if err != nil {
		return nil, err
	}
	return &commandSession{client: client, platform: target.Environment.SSH.Platform, fingerprint: fingerprint}, nil
}

type commandSession struct {
	client                *ssh.Client
	platform, fingerprint string
}

func (s *commandSession) HostKeyFingerprint() string { return s.fingerprint }
func (s *commandSession) Close() error               { return s.client.Close() }
func (s *commandSession) Run(ctx context.Context, command environmentport.Command, stdout, stderr io.Writer) (environmentport.CommandOutcome, error) {
	var result environmentport.CommandOutcome
	script, err := genericCommand(s.platform, command.Workdir, command.Argv)
	if err != nil {
		return result, err
	}
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stop()
	session, err := s.client.NewSession()
	if err != nil {
		return result, err
	}
	defer func() { _ = session.Close() }()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	session.Stdin = bytes.NewReader(command.Stdin)
	session.Stdout, session.Stderr = stdout, stderr
	result.Sent = true
	if err := session.Start(script); err != nil {
		return result, errors.New("SSH exec request failed; result unknown")
	}
	err = session.Wait()
	var exit *ssh.ExitError
	if err == nil {
		code := int32(0)
		result.ExitCode = &code
		return result, nil
	}
	if errors.As(err, &exit) {
		code := int32(exit.ExitStatus())
		if code >= 0 {
			result.ExitCode = &code
			return result, nil
		}
	}
	return result, errors.New("SSH command transport lost")
}

// Unlike deployment commands this preserves every argv element literally.
func genericCommand(platform, workdir string, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("program is required")
	}
	parts := make([]string, len(argv))
	switch platform {
	case model.EnvironmentPlatformLinux:
		for i, arg := range argv {
			parts[i] = posixQuote(arg)
		}
		return "cd " + posixRemotePath(workdir) + " && exec " + strings.Join(parts, " "), nil
	case model.EnvironmentPlatformWindows:
		for i, arg := range argv {
			parts[i] = powerShellQuote(arg)
		}
		// Windows PowerShell's native argument binder drops empty arguments and
		// embedded quotes. Launch native executables with an explicit CRT argv
		// command line, inheriting the SSH standard handles without text conversion.
		nativeArgs := make([]string, 0, len(argv)-1)
		for _, arg := range argv[1:] {
			nativeArgs = append(nativeArgs, windowsArg(arg))
		}
		script := "$ErrorActionPreference = 'Stop'; try { Set-Location -LiteralPath " + powerShellRemotePath(workdir) +
			"; $command = Get-Command -Name ([System.Management.Automation.WildcardPattern]::Escape(" + parts[0] + ")) -ErrorAction Stop; " +
			"if ($command.CommandType -eq 'Application') { $info = New-Object System.Diagnostics.ProcessStartInfo; $info.FileName = $command.Source; $info.Arguments = " + powerShellQuote(strings.Join(nativeArgs, " ")) +
			"; $info.WorkingDirectory = (Get-Location).ProviderPath; $info.UseShellExecute = $false; $process = [System.Diagnostics.Process]::Start($info); $process.WaitForExit(); $code = $process.ExitCode; $process.Dispose(); exit $code }; $global:LASTEXITCODE = 0; & " + parts[0]
		if len(parts) > 1 {
			script += " @(" + strings.Join(parts[1:], ",") + ")"
		}
		script += "; if (-not $?) { if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; exit 1 }; exit $LASTEXITCODE } catch { [Console]::Error.WriteLine($_.Exception.Message); exit 1 }"
		return powerShellEncodedCommand(script), nil
	default:
		return "", errors.New("unsupported environment platform")
	}
}

// Quote one Windows CRT argv element, including empty strings and trailing '\'.
func windowsArg(value string) string {
	var result strings.Builder
	result.WriteByte('"')
	slashes := 0
	for _, r := range value {
		if r == '\\' {
			slashes++
			continue
		}
		if r == '"' {
			result.WriteString(strings.Repeat("\\", slashes*2+1))
		} else {
			result.WriteString(strings.Repeat("\\", slashes))
		}
		slashes = 0
		result.WriteRune(r)
	}
	result.WriteString(strings.Repeat("\\", slashes*2))
	result.WriteByte('"')
	return result.String()
}
