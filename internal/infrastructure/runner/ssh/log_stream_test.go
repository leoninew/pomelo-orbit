package sshrunner

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestStreamAtEnvironmentRootWithoutComposeConfiguration(t *testing.T) {
	target, server := startSessionServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var output bytes.Buffer
	err := NewRuntime().StreamAtEnvironmentRoot(ctx, target, &output, "docker", "compose", "-p", "demo-default", "logs", "--follow")
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != "ok" || server.sftpClients.Load() != 0 {
		t.Fatalf("stream output = %q, SFTP clients = %d", output.String(), server.sftpClients.Load())
	}
}
