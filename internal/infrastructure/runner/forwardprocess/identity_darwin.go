package forwardprocess

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func processCreated(pid int) (string, error) {
	if pid < 1 {
		return "", errProcessExited
	}
	process, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err == unix.ESRCH {
		return "", errProcessExited
	}
	if err != nil {
		return "", err
	}
	if process == nil || process.Proc.P_pid != int32(pid) {
		return "", errProcessExited
	}
	return fmt.Sprintf("%d:%d", process.Proc.P_starttime.Sec, process.Proc.P_starttime.Usec), nil
}
