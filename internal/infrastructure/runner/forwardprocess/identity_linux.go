package forwardprocess

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

func processCreated(pid int) (string, error) {
	if pid < 1 {
		return "", errProcessExited
	}
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if errors.Is(err, os.ErrNotExist) {
		return "", errProcessExited
	}
	if err != nil {
		return "", err
	}
	end := strings.LastIndexByte(string(data), ')')
	if end < 0 {
		return "", errors.New("invalid process stat")
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 {
		return "", errors.New("invalid process stat")
	}
	if fields[0] == "Z" {
		return "", errProcessExited
	}
	return fields[19], nil
}
