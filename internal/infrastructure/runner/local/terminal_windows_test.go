package localrunner

import (
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
	countHandles := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetProcessHandleCount")
	count := func() uint32 {
		t.Helper()
		var count uint32
		result, _, err := countHandles.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&count)))
		if result == 0 {
			t.Fatal(err)
		}
		return count
	}
	// Warm up Go's process and pipe bookkeeping before taking a baseline.
	session, _ := startTestLocalTerminal(t, t.TempDir())
	_ = session.Close()
	before := count()
	for range 8 {
		session, _ := startTestLocalTerminal(t, t.TempDir())
		_ = session.Close()
	}
	if after := count(); after > before+2 {
		t.Fatalf("terminal leaked Windows handles: before=%d after=%d", before, after)
	}
}
