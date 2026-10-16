package main

import (
	"bytes"
	"testing"
)

func TestCommandHelpDoesNotRequireConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"--help"}, {"serve", "--help"}, {"worker", "--help"}, {"mcp", "--help"}, {"cli", "env", "--help"}, {"cli", "env", "forward", "start", "--help"}} {
		command := newRootCommand()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(&output)
		command.SetArgs(args)
		if err := command.Execute(); err != nil || output.Len() == 0 {
			t.Fatalf("args=%v output=%q err=%v", args, output.String(), err)
		}
	}
}
