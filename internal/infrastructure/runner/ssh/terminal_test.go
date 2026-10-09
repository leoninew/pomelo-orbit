package sshrunner

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"golang.org/x/crypto/ssh"
)

type testTerminalPty struct {
	Term   string
	Cols   uint32
	Rows   uint32
	Width  uint32
	Height uint32
	Modes  string
}

func terminalSSHServer(t *testing.T) (environmentport.Target, <-chan testTerminalPty, <-chan [2]uint32) {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	keyBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	ptyRequests := make(chan testTerminalPty, 4)
	windowChanges := make(chan [2]uint32, 4)
	config := &ssh.ServerConfig{PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if conn.User() != "managed-user" || !bytes.Equal(key.Marshal(), signer.PublicKey().Marshal()) {
			return nil, errors.New("unexpected SSH identity")
		}
		return nil, nil
	}}
	config.AddHostKey(signer)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, channels, requests, err := ssh.NewServerConn(conn, config)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(requests)
				for newChannel := range channels {
					if newChannel.ChannelType() != "session" {
						_ = newChannel.Reject(ssh.UnknownChannelType, "session required")
						continue
					}
					channel, requests, err := newChannel.Accept()
					if err != nil {
						return
					}
					go func() {
						defer func() { _ = channel.Close() }()
						for request := range requests {
							switch request.Type {
							case "pty-req":
								var dimensions testTerminalPty
								if ssh.Unmarshal(request.Payload, &dimensions) != nil {
									return
								}
								ptyRequests <- dimensions
								_ = request.Reply(true, nil)
							case "window-change":
								var dimensions struct{ Cols, Rows, Width, Height uint32 }
								if ssh.Unmarshal(request.Payload, &dimensions) != nil {
									return
								}
								windowChanges <- [2]uint32{dimensions.Cols, dimensions.Rows}
							case "shell":
								_ = request.Reply(true, nil)
								go func() {
									_, _ = channel.Write([]byte("host-shell-ready"))
									buffer := make([]byte, 1024)
									for {
										n, err := channel.Read(buffer)
										if err != nil {
											return
										}
										if bytes.Contains(buffer[:n], []byte{4}) {
											_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Code uint32 }{7}))
											_ = channel.Close()
											return
										}
										_, _ = channel.Write(buffer[:n])
									}
								}()
							default:
								_ = request.Reply(false, nil)
							}
						}
					}()
				}
			}()
		}
	}()
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	target := environmentport.Target{
		Environment: model.Environment{TargetType: model.EnvironmentTargetTypeSSH,
			SSH: &model.EnvironmentSSHTarget{Host: host, Port: portNumber, Username: "managed-user",
				HostKeyFingerprint: ssh.FingerprintSHA256(signer.PublicKey())}},
		PrivateKey: &environmentdto.DeploymentSSHPrivateKey{PrivateKey: string(pem.EncodeToMemory(keyBlock))},
	}
	return target, ptyRequests, windowChanges
}

func TestSSHTerminalUsesPinnedManagedIdentityAndPTY(t *testing.T) {
	target, ptyRequests, windowChanges := terminalSSHServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := NewRuntime().StartTerminal(ctx, target, 100, 30)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	exit := make(chan int, 1)
	go func() { code, _ := session.Wait(); exit <- code }()
	pty := <-ptyRequests
	if pty.Term != "xterm-256color" || pty.Cols != 100 || pty.Rows != 30 {
		t.Fatalf("PTY = %+v", pty)
	}
	ready := make([]byte, len("host-shell-ready"))
	if _, err := io.ReadFull(session.Output(), ready); err != nil || string(ready) != "host-shell-ready" {
		t.Fatalf("shell ready err=%v", err)
	}
	if _, err := session.Input().Write([]byte("host-command\n")); err != nil {
		t.Fatal(err)
	}
	output := make([]byte, len("host-command\n"))
	if _, err := io.ReadFull(session.Output(), output); err != nil || string(output) != "host-command\n" {
		t.Fatalf("shell output err=%v", err)
	}
	if err := session.Resize(120, 40); err != nil {
		t.Fatal(err)
	}
	select {
	case size := <-windowChanges:
		if size != [2]uint32{120, 40} {
			t.Fatalf("window = %v", size)
		}
	case <-ctx.Done():
		t.Fatal("SSH window change missing")
	}
	if _, err := session.Input().Write([]byte{4}); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-exit:
		if code != 7 {
			t.Fatalf("exit = %d", code)
		}
	case <-ctx.Done():
		t.Fatal("SSH shell did not exit")
	}
	untrusted := target
	sshTarget := *target.Environment.SSH
	untrusted.Environment.SSH = &sshTarget
	untrusted.Environment.SSH.HostKeyFingerprint = "SHA256:untrusted"
	if _, err := NewRuntime().StartTerminal(ctx, untrusted, 80, 24); err == nil || !strings.Contains(err.Error(), "host key fingerprint mismatch") {
		t.Fatalf("untrusted host key error = %v", err)
	}
}

func TestSSHTerminalClosesWithUnreadOutput(t *testing.T) {
	target, _, _ := terminalSSHServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := NewRuntime().StartTerminal(ctx, target, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	closed := make(chan struct{})
	go func() { _ = session.Close(); close(closed) }()
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal("closing SSH with unread output blocked")
	}
	waited := make(chan struct{})
	go func() { _, _ = session.Wait(); close(waited) }()
	select {
	case <-waited:
	case <-ctx.Done():
		t.Fatal("SSH Wait did not complete after Close")
	}
}

func TestSSHTerminalObservesFirstHostKeyOnAuthenticatedConnection(t *testing.T) {
	target, _, _ := terminalSSHServer(t)
	expected := target.Environment.SSH.HostKeyFingerprint
	target.Environment.SSH.HostKeyFingerprint = ""
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := NewRuntime().StartTerminal(ctx, target, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	if session.(environmentport.TerminalHostKey).HostKeyFingerprint() != expected {
		t.Fatal("terminal did not report the key from its SSH connection")
	}
	_ = session.Close()
	target.Environment.SSH.Username = "unauthorized-user"
	if _, err := NewRuntime().StartTerminal(ctx, target, 80, 24); err == nil {
		t.Fatal("terminal accepted first host key without managed key authentication")
	}
}
