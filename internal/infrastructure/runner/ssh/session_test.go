package sshrunner

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sessionServer struct {
	connections atomic.Int32
	active      atomic.Int32
	sftpClients atomic.Int32
	holdSFTP    atomic.Bool
	blocked     chan struct{}
}

func startSessionServer(t *testing.T) (environmentport.Target, *sessionServer) {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(key, "")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "workspace"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "workspace", "fixture.txt"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &sessionServer{blocked: make(chan struct{}, 4)}
	config := &ssh.ServerConfig{NoClientAuth: true}
	config.AddHostKey(signer)
	var connections sync.Map
	var workers sync.WaitGroup
	accepted := make(chan struct{})
	go func() {
		defer close(accepted)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			connections.Store(conn, true)
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer connections.Delete(conn)
				defer func() { _ = conn.Close() }()
				sshConn, channels, requests, err := ssh.NewServerConn(conn, config)
				if err != nil {
					return
				}
				server.connections.Add(1)
				server.active.Add(1)
				defer server.active.Add(-1)
				go ssh.DiscardRequests(requests)
				for newChannel := range channels {
					channel, requests, err := newChannel.Accept()
					if err != nil {
						return
					}
					workers.Add(1)
					go func() {
						defer workers.Done()
						defer func() { _ = channel.Close() }()
						for request := range requests {
							switch request.Type {
							case "subsystem":
								if server.holdSFTP.Load() {
									server.blocked <- struct{}{}
									_ = sshConn.Wait()
									return
								}
								_ = request.Reply(true, nil)
								server.sftpClients.Add(1)
								files, err := sftp.NewServer(channel, sftp.WithServerWorkingDirectory(root))
								if err == nil {
									_ = files.Serve()
									_ = files.Close()
								}
								return
							case "exec":
								_ = request.Reply(true, nil)
								var command struct{ Command string }
								_ = ssh.Unmarshal(request.Payload, &command)
								if strings.Contains(command.Command, "block") {
									server.blocked <- struct{}{}
									_ = sshConn.Wait()
									return
								}
								_, _ = io.WriteString(channel, "ok")
								_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Code uint32 }{0}))
								return
							default:
								_ = request.Reply(false, nil)
							}
						}
					}()
				}
			}()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-accepted
		connections.Range(func(conn, _ any) bool { _ = conn.(net.Conn).Close(); return true })
		workers.Wait()
	})
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	portNumber, _ := strconv.Atoi(port)
	return environmentport.Target{
		Environment: model.Environment{Id: "test-env", ProjectId: "test-project", TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: "workspace",
			SSH: &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformLinux, Host: host, Port: portNumber, Username: "managed-user", HostKeyFingerprint: ssh.FingerprintSHA256(signer.PublicKey())}},
		PrivateKey: &environmentdto.DeploymentSSHPrivateKey{PrivateKey: string(pem.EncodeToMemory(block))},
	}, server
}

func TestRuntimeSessionReusesSSHAndSFTPAndClosesAtExit(t *testing.T) {
	target, server := startSessionServer(t)
	runtime := NewRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx, closeSession, err := runtime.OpenSession(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	for range 3 {
		body, err := runtime.ReadFile(ctx, target, target.Environment.WorkspaceRoot+"/fixture.txt")
		if err != nil || string(body) != "fixture" {
			t.Fatalf("read=%q err=%v", body, err)
		}
		output, err := runtime.QueryAtEnvironmentRoot(ctx, target, "echo", "hello")
		if err != nil || output != "ok" {
			t.Fatalf("query=%q err=%v", output, err)
		}
	}
	if server.connections.Load() != 1 || server.sftpClients.Load() != 1 {
		t.Fatalf("connections=%d SFTP clients=%d", server.connections.Load(), server.sftpClients.Load())
	}
	changed := target
	sshTarget := *target.Environment.SSH
	sshTarget.CredentialRevision++
	changed.Environment.SSH = &sshTarget
	if _, err := runtime.ReadFile(ctx, changed, target.Environment.WorkspaceRoot+"/fixture.txt"); err == nil {
		t.Fatal("session accepted a changed credential revision")
	}
	closeSession()
	for server.active.Load() != 0 {
		select {
		case <-ctx.Done():
			t.Fatal("session left an SSH connection open")
		case <-time.After(time.Millisecond):
		}
	}
	if _, err := runtime.ReadFile(ctx, target, target.Environment.WorkspaceRoot+"/fixture.txt"); err == nil {
		t.Fatal("closed session was reopened")
	}
}

func TestRuntimeSessionCancellationReconnectsForRecovery(t *testing.T) {
	for _, operation := range []string{"command", "SFTP startup"} {
		t.Run(operation, func(t *testing.T) {
			target, server := startSessionServer(t)
			runtime := NewRuntime()
			parent, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			ctx, closeSession, err := runtime.OpenSession(parent, target)
			if err != nil {
				t.Fatal(err)
			}
			defer closeSession()
			phase, cancel := context.WithCancel(ctx)
			defer cancel()
			finished := make(chan error, 1)
			server.holdSFTP.Store(operation == "SFTP startup")
			go func() {
				if operation == "command" {
					_, err := runtime.QueryAtEnvironmentRoot(phase, target, "block")
					finished <- err
				} else {
					_, err := runtime.ReadFile(phase, target, target.Environment.WorkspaceRoot+"/fixture.txt")
					finished <- err
				}
			}()
			select {
			case <-server.blocked:
			case <-parent.Done():
				t.Fatal("phase did not reach the remote operation")
			}
			cancel()
			select {
			case err := <-finished:
				if err == nil {
					t.Fatal("canceled operation reported success")
				}
			case <-parent.Done():
				t.Fatal("cancellation did not interrupt the remote operation")
			}
			server.holdSFTP.Store(false)
			recovery, stopRecovery := context.WithTimeout(context.WithoutCancel(phase), time.Second)
			defer stopRecovery()
			if body, err := runtime.ReadFile(recovery, target, target.Environment.WorkspaceRoot+"/fixture.txt"); err != nil || string(body) != "fixture" {
				t.Fatalf("recovery read=%q err=%v", body, err)
			}
			if server.connections.Load() != 2 {
				t.Fatalf("recovery reused a canceled connection: connections=%d", server.connections.Load())
			}
		})
	}
}

func TestRuntimeSessionDiscardsClosedSFTPTransport(t *testing.T) {
	target, server := startSessionServer(t)
	runtime := NewRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx, closeSession, err := runtime.OpenSession(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	if _, err := runtime.ReadFile(ctx, target, target.Environment.WorkspaceRoot+"/fixture.txt"); err != nil {
		t.Fatal(err)
	}
	scope := ctx.Value(sessionContextKey{}).(*runtimeSession)
	scope.mu.Lock()
	files := scope.files
	scope.mu.Unlock()
	_ = files.Close()
	for {
		scope.mu.Lock()
		discarded := scope.client == nil
		scope.mu.Unlock()
		if discarded {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("closed SFTP transport remained in the session")
		case <-time.After(time.Millisecond):
		}
	}
	if _, err := runtime.ReadFile(ctx, target, target.Environment.WorkspaceRoot+"/fixture.txt"); err != nil {
		t.Fatal(err)
	}
	if server.connections.Load() != 2 || server.sftpClients.Load() != 2 {
		t.Fatalf("connection=%d SFTP clients=%d", server.connections.Load(), server.sftpClients.Load())
	}
}
