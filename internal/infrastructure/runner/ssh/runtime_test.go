package sshrunner

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestCanceledWorkspaceDoesNotConnectWhileFileLockIsHeld(t *testing.T) {
	runtime := NewRuntime()
	target := environmentport.Target{Environment: model.Environment{Id: "environment-1", TargetType: model.EnvironmentTargetTypeSSH, SSH: &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformLinux}}}
	directory := "/workspace/api"
	unlock, err := runtime.lock(context.Background(), target.Environment.Id+"\x00"+directory)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- runtime.StageWorkspace(ctx, target, deploymentport.Workspace{Location: deploymentport.ServiceLocation{Code: "api", Directory: directory}, Compose: "services: {}\n"})
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled workspace returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("workspace waited for previous file publisher")
	}
}

func TestRuntimeUpdatesExistingCompose(t *testing.T) {
	target, _ := startSessionServer(t)
	runtime := NewRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx, closeSession, err := runtime.OpenSession(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	client, cleanup, err := runtime.openSFTP(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	directory, err := newSFTPPathResolver(client, target.Environment.SSH.Platform).resolve("~/workspace")
	if err != nil {
		t.Fatal(err)
	}
	composePath := path.Join(directory, "docker-compose.yml")
	file, err := client.Create(composePath)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.Write([]byte("services:\n  api:\n    image: api:old\n"))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("write existing Compose: %v, close: %v", writeErr, closeErr)
	}
	workspace := deploymentport.Workspace{
		Location: deploymentport.ServiceLocation{Code: "api", Directory: "~/workspace"},
		Compose:  "services: {}\n",
	}
	if err := runtime.StageWorkspace(ctx, target, workspace); err != nil {
		t.Fatal(err)
	}
	target.FileScope = directory
	if body, err := runtime.ReadFile(ctx, target, composePath); err != nil || string(body) != workspace.Compose {
		t.Fatalf("staged Compose=%q err=%v", body, err)
	}
	if body, err := runtime.ReadFile(ctx, target, path.Join(directory, "fixture.txt")); err != nil || string(body) != "fixture" {
		t.Fatalf("existing file=%q err=%v", body, err)
	}
}

type workspaceRenamerStub struct {
	files       map[string]bool
	failInstall bool
	posixCalls  int
}

func (stub *workspaceRenamerStub) Rename(oldPath, newPath string) error {
	if !stub.files[oldPath] {
		return os.ErrNotExist
	}
	if stub.files[newPath] || stub.failInstall && oldPath == "staged" && newPath == "current" {
		return os.ErrPermission
	}
	delete(stub.files, oldPath)
	stub.files[newPath] = true
	return nil
}

func (stub *workspaceRenamerStub) PosixRename(oldPath, newPath string) error {
	stub.posixCalls++
	delete(stub.files, oldPath)
	stub.files[newPath] = true
	return nil
}

func (stub *workspaceRenamerStub) Remove(filePath string) error {
	delete(stub.files, filePath)
	return nil
}

func TestCommitWorkspaceFileReplacesExistingWindowsFile(t *testing.T) {
	stub := &workspaceRenamerStub{files: map[string]bool{"staged": true, "current": true}}
	if err := commitWorkspaceFile(stub, model.EnvironmentPlatformWindows, "staged", "current"); err != nil {
		t.Fatalf("replace Windows file: %v", err)
	}
	if !stub.files["current"] || stub.files["staged"] || len(stub.files) != 1 || stub.posixCalls != 0 {
		t.Fatalf("unexpected files after replacement: %+v", stub.files)
	}
}

func TestCommitWorkspaceFileRestoresWindowsFileWhenReplacementFails(t *testing.T) {
	stub := &workspaceRenamerStub{files: map[string]bool{"staged": true, "current": true}, failInstall: true}
	if err := commitWorkspaceFile(stub, model.EnvironmentPlatformWindows, "staged", "current"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected replacement permission error, got %v", err)
	}
	if !stub.files["current"] || !stub.files["staged"] || len(stub.files) != 2 {
		t.Fatalf("previous file was not restored: %+v", stub.files)
	}
}

func TestCommitWorkspaceFileKeepsAtomicRenameOnLinux(t *testing.T) {
	stub := &workspaceRenamerStub{files: map[string]bool{"staged": true, "current": true}}
	if err := commitWorkspaceFile(stub, model.EnvironmentPlatformLinux, "staged", "current"); err != nil {
		t.Fatalf("replace Linux file: %v", err)
	}
	if stub.posixCalls != 1 || !stub.files["current"] || stub.files["staged"] {
		t.Fatalf("Linux replacement did not use POSIX rename: %+v", stub)
	}
}

func TestNormalizeSFTPHomePathForWindowsAndLinux(t *testing.T) {
	for _, testCase := range []struct {
		platform string
		input    string
		want     string
	}{
		{model.EnvironmentPlatformWindows, `/C:/Users/deploy`, `C:/Users/deploy`},
		{model.EnvironmentPlatformWindows, `C:\Users\deploy`, `C:/Users/deploy`},
		{model.EnvironmentPlatformLinux, `/C:/Users/deploy`, `/C:/Users/deploy`},
	} {
		if got := normalizeSFTPHomePath(testCase.platform, testCase.input); got != testCase.want {
			t.Fatalf("normalizeSFTPHomePath(%q, %q) = %q, want %q", testCase.platform, testCase.input, got, testCase.want)
		}
	}
}

func TestServiceDirUsesTargetPlatformPaths(t *testing.T) {
	runtime := NewRuntime()
	for _, testCase := range []struct {
		name, platform, root, want string
	}{
		{"Linux", model.EnvironmentPlatformLinux, "/opt/pomelo-orbit/data", "/opt/pomelo-orbit/data/deployment/traefik-default"},
		{"Windows", model.EnvironmentPlatformWindows, `D:\orbit\data`, "D:/orbit/data/deployment/traefik-default"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			target := environmentport.Target{Environment: model.Environment{
				TargetType:    model.EnvironmentTargetTypeSSH,
				WorkspaceRoot: testCase.root,
				SSH:           &model.EnvironmentSSHTarget{Platform: testCase.platform},
			}}
			got, err := runtime.ServiceDir(target, deploymentport.ServiceLocation{Code: "traefik-default", Directory: strings.TrimRight(target.Environment.WorkspaceRoot, "/") + "/deployment/traefik-default"})
			if err != nil {
				t.Fatalf("ComposeMountSourceDir returned error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("ComposeMountSourceDir = %q, want %q", got, testCase.want)
			}
		})
	}
}

type sftpRealPathStub string

func (stub sftpRealPathStub) RealPath(path string) (string, error) {
	if path != "." {
		return "", errors.New("unexpected SFTP path")
	}
	return string(stub), nil
}

type sftpCanonicalPathStub map[string]string

func (stub sftpCanonicalPathStub) RealPath(value string) (string, error) {
	if resolved, ok := stub[value]; ok {
		return resolved, nil
	}
	return "", os.ErrNotExist
}

func TestRemoteCanonicalPathsKeepFileOperationsInServiceScope(t *testing.T) {
	for _, tc := range []struct{ platform, root, alias, escaped string }{
		{"linux", "/srv/api", "/alias/api", "/srv/other"},
		{"windows", "D:/Services/api", "D:/Alias/api", "/D:/Services/other"},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			resolver := newSFTPPathResolver(sftpCanonicalPathStub{
				tc.root: tc.root, tc.alias: tc.root, tc.root + "/linked": tc.escaped,
			}, tc.platform)
			got, err := resolver.canonical(tc.alias + "/new/config")
			if err != nil || got != tc.root+"/new/config" {
				t.Fatalf("canonical=%s err=%v", got, err)
			}
			if _, err := resolver.scoped(tc.root, tc.root+"/new/config"); err != nil {
				t.Fatal(err)
			}
			if _, err := resolver.scoped(tc.root, tc.root+"/linked/config"); err == nil {
				t.Fatal("symlink escaped the service scope")
			}
		})
	}
}

func TestSFTPPathResolverExpandsRemoteHomeForMountSource(t *testing.T) {
	for _, testCase := range []struct {
		name, platform, home, want string
	}{
		{"Linux", model.EnvironmentPlatformLinux, "/home/deploy", "/home/deploy/orbit/deployment/traefik-default"},
		{"Windows", model.EnvironmentPlatformWindows, "/C:/Users/deploy", "C:/Users/deploy/orbit/deployment/traefik-default"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resolver := newSFTPPathResolver(sftpRealPathStub(testCase.home), testCase.platform)
			got, err := resolver.resolve("~/orbit/deployment/traefik-default")
			if err != nil {
				t.Fatalf("resolve remote home: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("resolved path = %q, want %q", got, testCase.want)
			}
		})
	}
}

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
