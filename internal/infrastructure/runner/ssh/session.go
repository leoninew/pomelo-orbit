package sshrunner

import (
	"context"
	"errors"
	"sync"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sessionContextKey struct{}

type runtimeSession struct {
	runtime *Runtime
	target  environmentport.Target
	mu      sync.Mutex
	client  *ssh.Client
	files   *sftp.Client
	closed  bool
}

func (r *Runtime) OpenSession(ctx context.Context, target environmentport.Target) (context.Context, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if !target.Environment.IsSSH() || target.PrivateKey == nil {
		return nil, nil, errors.New("SSH runtime session received an invalid target")
	}
	sshTarget := *target.Environment.SSH
	target.Environment.SSH = &sshTarget
	privateKey := *target.PrivateKey
	target.PrivateKey = &privateKey
	session := &runtimeSession{runtime: r, target: target}
	return context.WithValue(ctx, sessionContextKey{}, session), session.close, nil
}

func (s *runtimeSession) matches(r *Runtime, target environmentport.Target) bool {
	a, b := s.target.Environment, target.Environment
	return s.runtime == r && a.Id == b.Id && a.ProjectId == b.ProjectId && a.TargetRevision == b.TargetRevision && a.WorkspaceRoot == b.WorkspaceRoot && b.IsSSH() && *a.SSH == *b.SSH
}

func (s *runtimeSession) acquire(ctx context.Context) (*ssh.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.closed {
		return nil, errors.New("SSH runtime session is closed")
	}
	if s.client == nil {
		client, err := s.runtime.dialSSH(ctx, s.target)
		if err != nil {
			return nil, err
		}
		s.client = client
		go func() {
			_ = client.Wait()
			s.discard(client)
		}()
	}
	return s.client, nil
}

func (s *runtimeSession) fileClient(client *ssh.Client) (*sftp.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.client != client {
		return nil, errors.New("SSH runtime session connection was closed")
	}
	if s.files == nil {
		files, err := sftp.NewClient(client)
		if err != nil {
			return nil, err
		}
		s.files = files
		go func() {
			_ = files.Wait()
			s.discard(client)
		}()
	}
	return s.files, nil
}

func (s *runtimeSession) discard(client *ssh.Client) {
	// Close before taking the mutex so cancellation can interrupt SFTP startup.
	_ = client.Close()
	s.mu.Lock()
	var files *sftp.Client
	if s.client == client {
		files, s.files, s.client = s.files, nil, nil
	}
	s.mu.Unlock()
	if files != nil {
		_ = files.Close()
	}
}

func (s *runtimeSession) close() {
	s.mu.Lock()
	s.closed = true
	client := s.client
	s.mu.Unlock()
	if client != nil {
		s.discard(client)
	}
}
