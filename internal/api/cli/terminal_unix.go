//go:build !windows

package cli

import "os"

func enableTerminalOutput(*os.File) (func(), error) { return func() {}, nil }
