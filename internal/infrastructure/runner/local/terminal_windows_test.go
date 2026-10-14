package localrunner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func terminalChildCommand(pidPath string) string {
	systemDirectory, _ := windows.GetSystemDirectory()
	shell := filepath.Join(systemDirectory, "WindowsPowerShell", "v1.0", "powershell.exe")
	return "& '" + shell + "' -NoLogo -NoProfile -Command '$PID | Set-Content -LiteralPath ''" + strings.ReplaceAll(pidPath, "'", "''''") + "''; Start-Sleep -Seconds 60'"
}

func terminalTestProcessAlive(pid int) bool {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(handle) }()
	status, _ := windows.WaitForSingleObject(handle, 0)
	return status == uint32(windows.WAIT_TIMEOUT)
}

func TestLocalTerminalReleasesWindowsHandles(t *testing.T) {
	session, _ := startTestLocalTerminal(t, t.TempDir())
	terminal := session.(*localTerminal)
	process := terminal.process.(*windowsTerminalProcess)
	// Query the owned resource types: process-wide counts include runtime
	// handles, and a closed handle value may be reused for an event or pipe.
	queries := map[string]func() error{
		"process": func() error {
			var code uint32
			return windows.GetExitCodeProcess(process.process, &code)
		},
		"job": func() error {
			var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
			return windows.QueryInformationJobObject(process.job, windows.JobObjectExtendedLimitInformation,
				uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil)
		},
	}
	for name, query := range queries {
		if err := query(); err != nil {
			t.Fatalf("%s was not open: %v", name, err)
		}
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	for name, query := range queries {
		if err := query(); !errors.Is(err, windows.ERROR_INVALID_HANDLE) {
			t.Fatalf("%s remains open after Close: %v", name, err)
		}
	}
	if _, err := terminal.reader.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("terminal output pipe remains open: %v", err)
	}
	if _, err := session.Input().Write([]byte("x")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("terminal input pipe remains open: %v", err)
	}
}
