package localrunner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStreamAtEnvironmentRootWithoutComposeConfiguration(t *testing.T) {
	if os.Getenv("POMELO_ORBIT_LOG_STREAM_HELPER") == "1" {
		fmt.Println("container output")
		os.Exit(0)
	}
	t.Setenv("POMELO_ORBIT_LOG_STREAM_HELPER", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "workspace")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var output bytes.Buffer
	err = NewRuntime(nil).StreamAtEnvironmentRoot(ctx, localTarget(root), &output, executable, "-test.run=^TestStreamAtEnvironmentRootWithoutComposeConfiguration$")
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != "container output\n" {
		t.Fatalf("stream output = %q", output.String())
	}
}
