package sshrunner

import (
	"encoding/base64"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRemoteCommandUsesWindowsDockerDesktopCLI(t *testing.T) {
	command, display, err := remoteCommand(model.EnvironmentPlatformWindows, `C:\orbit workspace\service`, "docker", "compose", "-p", "project name", "up", "-d")
	if err != nil {
		t.Fatalf("remoteCommand returned error: %v", err)
	}
	script := decodePowerShellEncodedCommand(t, command)
	for _, expected := range []string{"$ProgressPreference = 'SilentlyContinue'", "Set-Location -LiteralPath", "docker.exe", "'project name'"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("script %q does not contain %q", script, expected)
		}
	}
	if strings.Contains(script, "wsl.exe -d docker-desktop") {
		t.Fatalf("command must use the host Docker Desktop CLI: %q", script)
	}
	if display != "docker compose -p project name up -d" {
		t.Fatalf("display = %q", display)
	}
}

func TestRemoteCommandQuotesLinuxArguments(t *testing.T) {
	command, _, err := remoteCommand(model.EnvironmentPlatformLinux, "/srv/orbit/service one", "docker", "compose", "-p", "project'name", "ps")
	if err != nil {
		t.Fatalf("remoteCommand returned error: %v", err)
	}
	if !strings.HasPrefix(command, "sh -lc ") || !strings.Contains(command, "/srv/orbit/service one") {
		t.Fatalf("linux command is not safely structured: %q", command)
	}
	if quoted := posixQuote("project'name"); quoted != `'project'"'"'name'` {
		t.Fatalf("posixQuote = %q", quoted)
	}
}

func TestRemoteCommandResolvesSSHHomeWorkspace(t *testing.T) {
	linuxCommand, _, err := remoteCommand(model.EnvironmentPlatformLinux, "~/.pomelo-orbit/service", "docker", "compose", "config")
	if err != nil {
		t.Fatalf("remoteCommand returned error: %v", err)
	}
	if !strings.Contains(linuxCommand, "$HOME") || !strings.Contains(linuxCommand, "/.pomelo-orbit/service") {
		t.Fatalf("Linux command does not resolve SSH home: %q", linuxCommand)
	}

	windowsCommand, _, err := remoteCommand(model.EnvironmentPlatformWindows, "~/.pomelo-orbit/service", "docker", "compose", "config")
	if err != nil {
		t.Fatalf("remoteCommand returned error: %v", err)
	}
	windowsScript := decodePowerShellEncodedCommand(t, windowsCommand)
	if !strings.Contains(windowsScript, "Set-Location -LiteralPath (Join-Path -Path $HOME -ChildPath '.pomelo-orbit/service')") {
		t.Fatalf("Windows command does not resolve SSH home: %q", windowsScript)
	}
}

func TestRemoteCommandUsesWindowsCurlExecutable(t *testing.T) {
	command, _, err := remoteCommand(model.EnvironmentPlatformWindows, `C:\orbit\gateway`, "curl", "-fsS", "http://127.0.0.1:8080/api/overview")
	if err != nil {
		t.Fatalf("remoteCommand returned error: %v", err)
	}
	if !strings.Contains(decodePowerShellEncodedCommand(t, command), "curl.exe") {
		t.Fatalf("Windows Traefik command must invoke curl.exe: %q", command)
	}
}

func TestRemoteQueryDiagnosticKeepsSuccessfulStdoutClean(t *testing.T) {
	if got := remoteQueryDiagnostic(`[{"name":"router"}]`, "diagnostic"); got != "[{\"name\":\"router\"}]\ndiagnostic" {
		t.Fatalf("query diagnostic = %q", got)
	}
	if got := remoteQueryDiagnostic(`[{"name":"router"}]`, ""); got != "[{\"name\":\"router\"}]" {
		t.Fatalf("stdout-only result = %q", got)
	}
}

func TestPowerShellEncodedCommandUsesUTF16LE(t *testing.T) {
	const script = "$ErrorActionPreference = 'Stop'; Write-Output 'ready'"
	const prefix = "$ProgressPreference = 'SilentlyContinue'; "
	if decoded := decodePowerShellEncodedCommand(t, powerShellEncodedCommand(script)); decoded != prefix+script {
		t.Fatalf("decoded script = %q", decoded)
	}
}

func decodePowerShellEncodedCommand(t *testing.T, command string) string {
	t.Helper()
	if !strings.HasPrefix(command, powerShellEncodedCommandPrefix) {
		t.Fatalf("expected encoded PowerShell command, got %q", command)
	}
	encoded := strings.TrimPrefix(command, powerShellEncodedCommandPrefix)
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode PowerShell command: %v", err)
	}
	if len(bytes)%2 != 0 {
		t.Fatalf("encoded PowerShell command has odd UTF-16LE length: %d", len(bytes))
	}
	codeUnits := make([]uint16, len(bytes)/2)
	for index := range codeUnits {
		codeUnits[index] = uint16(bytes[index*2]) | uint16(bytes[index*2+1])<<8
	}
	return string(utf16.Decode(codeUnits))
}
