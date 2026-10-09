package localrunner

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/aymanbagabas/go-pty"
	"golang.org/x/sys/windows"
)

type windowsTerminalProcess struct {
	process windows.Handle
	job     windows.Handle
	once    sync.Once
}

func startTerminalProcess(terminal pty.Pty, directory string) (terminalProcess, error) {
	systemDirectory, err := windows.GetSystemDirectory()
	if err != nil {
		return nil, err
	}
	shell := filepath.Join(systemDirectory, "WindowsPowerShell", "v1.0", "powershell.exe")
	application, err := windows.UTF16PtrFromString(shell)
	if err != nil {
		return nil, err
	}
	commandLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine([]string{shell, "-NoLogo", "-NoProfile"}))
	if err != nil {
		return nil, err
	}
	cwd, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return nil, err
	}
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	attributes, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	defer attributes.Delete()
	// go-pty owns ConPTY. Its Cmd.Start leaks CreateProcess's original process
	// handle in v0.2.3, so this adapter owns the process and thread handles.
	console := terminal.Fd()
	//nolint:govet // Win32 requires the HPCON handle value cast to LPVOID here.
	if err := attributes.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, unsafe.Pointer(console), unsafe.Sizeof(console)); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	startup := windows.StartupInfoEx{}
	startup.Flags = windows.STARTF_USESTDHANDLES
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.ProcThreadAttributeList = attributes.List()
	var process windows.ProcessInformation
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_SUSPENDED)
	// A nil environment inherits Orbit's complete environment.
	if err := windows.CreateProcess(application, commandLine, nil, nil, false, flags, nil, cwd, &startup.StartupInfo, &process); err != nil {
		_ = windows.CloseHandle(job)
		return nil, err
	}
	defer func() { _ = windows.CloseHandle(process.Thread) }()
	cleanup := func() {
		_ = windows.TerminateProcess(process.Process, 1)
		_, _ = windows.WaitForSingleObject(process.Process, windows.INFINITE)
		_ = windows.CloseHandle(process.Process)
		_ = windows.CloseHandle(job)
	}
	if err := windows.AssignProcessToJobObject(job, process.Process); err != nil {
		cleanup()
		return nil, err
	}
	if _, err := windows.ResumeThread(process.Thread); err != nil {
		cleanup()
		return nil, err
	}
	return &windowsTerminalProcess{process: process.Process, job: job}, nil
}

func (p *windowsTerminalProcess) wait() (int, error) {
	defer func() { _ = windows.CloseHandle(p.process) }()
	if _, err := windows.WaitForSingleObject(p.process, windows.INFINITE); err != nil {
		return 1, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(p.process, &code); err != nil {
		return 1, err
	}
	return int(code), nil
}

func (p *windowsTerminalProcess) stop() {
	p.once.Do(func() { _ = windows.CloseHandle(p.job) })
}

func terminalOutputReader(terminal pty.Pty) (io.ReadCloser, error) {
	// Transfer output ownership before starting reads. Closing an active original
	// pipe later would cancel reads on its duplicate too and truncate tail output.
	var handle windows.Handle
	process := windows.CurrentProcess()
	output := terminal.(pty.ConPty).OutputPipe()
	if err := windows.DuplicateHandle(process, windows.Handle(output.Fd()), process, &handle, 0, false, windows.DUPLICATE_SAME_ACCESS); err != nil {
		return nil, err
	}
	if err := output.Close(); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	return os.NewFile(uintptr(handle), "terminal-output"), nil
}

func finishTerminalOutput(s *localTerminal) { s.closePty() }

func terminalOutputEnded(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) || errors.Is(err, windows.ERROR_BROKEN_PIPE)
}
