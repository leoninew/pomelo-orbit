package sshrunner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// Component verification uses real local TCP sockets and SSH channels. The
// in-process SSH server replies only after EOF to exercise TCP half-close.
func TestForwardStreamsMultipleConnectionsAndClosesWhenSSHLost(t *testing.T) {
	target, server := startSessionServer(t)
	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	localPort := reservation.Addr().(*net.TCPAddr).Port
	_ = reservation.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	session, err := NewRuntime().OpenForward(ctx, target, localPort, 5432)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	if session.HostKeyFingerprint() != target.Environment.SSH.HostKeyFingerprint {
		t.Fatal("forward lost authenticated host identity")
	}
	if err := session.Listen(); err != nil {
		t.Fatal(err)
	}
	address := fmt.Sprintf("127.0.0.1:%d", localPort)
	var streams sync.WaitGroup
	for _, input := range [][]byte{{0, 255, 'a'}, []byte("another connection")} {
		streams.Go(func() {
			connection, err := net.DialTimeout("tcp4", address, time.Second)
			if err != nil {
				t.Error(err)
				return
			}
			defer func() { _ = connection.Close() }()
			_ = connection.SetDeadline(time.Now().Add(3 * time.Second))
			if _, err := connection.Write(input); err != nil {
				t.Error(err)
				return
			}
			if err := connection.(*net.TCPConn).CloseWrite(); err != nil {
				t.Error(err)
				return
			}
			output, err := io.ReadAll(connection)
			if err != nil || !bytes.Equal(output, append([]byte("reply:"), input...)) {
				t.Errorf("half-close output=%q err=%v", output, err)
			}
		})
	}
	streams.Wait()
	for range 2 {
		select {
		case destination := <-server.forwards:
			if destination.Host != "127.0.0.1" || destination.Port != 5432 {
				t.Fatalf("forward reached an unexpected destination: %+v", destination)
			}
		case <-ctx.Done():
			t.Fatal("forward did not reach SSH server")
		}
	}
	active, err := net.DialTimeout("tcp4", address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = active.Close() }()
	server.disconnect()
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("lost SSH left forward running")
	}
	_ = active.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := active.Read(make([]byte, 1)); err == nil {
		t.Fatal("active forward stream remained open")
	}
	connection, err := net.DialTimeout("tcp4", address, time.Second)
	if err == nil {
		_ = connection.Close()
		t.Fatal("lost SSH left local listener open")
	}
}
