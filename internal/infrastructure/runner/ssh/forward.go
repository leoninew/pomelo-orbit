package sshrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"golang.org/x/crypto/ssh"
)

var _ environmentport.ForwardRunner = (*Runtime)(nil)

type forwardSession struct {
	client                *ssh.Client
	fingerprint           string
	localPort, remotePort int
	ctx                   context.Context
	cancel                context.CancelFunc
	mu                    sync.Mutex
	listener              net.Listener
	connections           map[net.Conn]struct{}
	closed                bool
	once                  sync.Once
	done                  chan struct{}
	err                   error
}

func (r *Runtime) OpenForward(ctx context.Context, target environmentport.Target, localPort, remotePort int) (environmentport.ForwardSession, error) {
	if localPort < 1 || localPort > 65535 || remotePort < 1 || remotePort > 65535 {
		return nil, errors.New("forward ports must be 1–65535")
	}
	var fingerprint string
	client, err := r.dialSSHWithHostKey(ctx, target, &fingerprint)
	if err != nil {
		return nil, err
	}
	lifetime, cancel := context.WithCancel(context.Background())
	return &forwardSession{client: client, fingerprint: fingerprint, localPort: localPort, remotePort: remotePort,
		ctx: lifetime, cancel: cancel, done: make(chan struct{}), connections: make(map[net.Conn]struct{})}, nil
}

func (s *forwardSession) HostKeyFingerprint() string { return s.fingerprint }

func (s *forwardSession) Listen() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.listener != nil {
		return errors.New("forward is closed or already listening")
	}
	listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(s.localPort)))
	if err != nil {
		return fmt.Errorf("listen on local forward port: %w", err)
	}
	s.listener = listener
	go s.accept(listener)
	go func() { s.finish(s.client.Wait()) }()
	go s.keepAlive()
	return nil
}

func (s *forwardSession) accept(listener net.Listener) {
	for {
		connection, err := listener.Accept()
		if err != nil {
			s.finish(err)
			return
		}
		if !s.track(connection) {
			return
		}
		go s.forward(connection)
	}
}

func (s *forwardSession) track(connection net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		_ = connection.Close()
		return false
	}
	s.connections[connection] = struct{}{}
	return true
}

func (s *forwardSession) release(connection net.Conn) {
	_ = connection.Close()
	s.mu.Lock()
	delete(s.connections, connection)
	s.mu.Unlock()
}

func (s *forwardSession) forward(local net.Conn) {
	defer s.release(local)
	ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
	remote, err := s.client.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(s.remotePort)))
	cancel()
	if err != nil || !s.track(remote) {
		return
	}
	defer s.release(remote)
	done := make(chan struct{}, 1)
	go func() { forwardCopy(remote, local); done <- struct{}{} }()
	forwardCopy(local, remote)
	<-done
}

func forwardCopy(destination, source net.Conn) {
	if _, err := io.Copy(destination, source); err != nil {
		_ = destination.Close()
		_ = source.Close()
		return
	}
	if writer, ok := destination.(interface{ CloseWrite() error }); ok {
		_ = writer.CloseWrite()
	}
}

func (s *forwardSession) keepAlive() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			reply := make(chan error, 1)
			go func() { _, _, err := s.client.SendRequest("keepalive@openssh.com", true, nil); reply <- err }()
			timer := time.NewTimer(5 * time.Second)
			select {
			case err := <-reply:
				timer.Stop()
				if err != nil {
					s.finish(errors.New("SSH forward connection lost"))
					return
				}
			case <-timer.C:
				s.finish(errors.New("SSH forward keepalive timed out"))
				return
			case <-s.ctx.Done():
				timer.Stop()
				return
			}
		}
	}
}

func (s *forwardSession) finish(err error) {
	s.once.Do(func() {
		s.cancel()
		s.mu.Lock()
		s.closed, s.err = true, err
		listener := s.listener
		connections := make([]net.Conn, 0, len(s.connections))
		for connection := range s.connections {
			connections = append(connections, connection)
		}
		s.mu.Unlock()
		if listener != nil {
			_ = listener.Close()
		}
		for _, connection := range connections {
			_ = connection.Close()
		}
		_ = s.client.Close()
		close(s.done)
	})
}

func (s *forwardSession) Wait() error  { <-s.done; return s.err }
func (s *forwardSession) Close() error { s.finish(nil); return nil }
