package authsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const mcpGrantTTL = 2 * time.Minute

type mcpGrantStore struct {
	mu     sync.Mutex
	grants map[[32]byte]mcpGrantRecord
}

type mcpGrantRecord struct {
	userID    string
	expiresAt time.Time
}

func newMCPGrantStore() *mcpGrantStore {
	return &mcpGrantStore{grants: make(map[[32]byte]mcpGrantRecord)}
}

// IssueMCPGrant creates a short-lived, single-use code. The raw code is only
// returned to the authenticated browser and is never stored or logged.
func (s Service) IssueMCPGrant(_ context.Context, user model.User, input authdto.MCPGrantInput) (authdto.MCPGrant, error) {
	if err := ValidateMCPGrantInput(input); err != nil {
		return authdto.MCPGrant{}, err
	}
	if strings.TrimSpace(user.Id) == "" || user.Status != "enabled" {
		return authdto.MCPGrant{}, apperror.New(apperror.KindUnauthorized, "Invalid token")
	}
	code, err := newMCPGrantCode()
	if err != nil {
		return authdto.MCPGrant{}, err
	}
	expiresAt := time.Now().UTC().Add(mcpGrantTTL)
	s.grants.put(code, mcpGrantRecord{userID: user.Id, expiresAt: expiresAt})
	return authdto.MCPGrant{Code: code, ExpiresAt: expiresAt}, nil
}

// ExchangeMCPGrant consumes a grant and returns a normal Orbit bearer token.
// The user is reloaded so a disabled or deleted account cannot use a grant
// issued before its state changed.
func (s Service) ExchangeMCPGrant(ctx context.Context, code string) (string, error) {
	userID, err := s.grants.consume(code, time.Now().UTC())
	if err != nil {
		return "", err
	}
	user, err := s.repo.UserById(ctx, userID)
	if err != nil || user.Status != "enabled" {
		return "", apperror.New(apperror.KindUnauthorized, "MCP authorization code is invalid or expired")
	}
	return s.tokens.Sign(user.Id, user.Username)
}

func ValidateMCPGrantInput(input authdto.MCPGrantInput) error {
	if strings.TrimSpace(input.State) == "" || len(input.State) < 32 {
		return apperror.New(apperror.KindValidation, "MCP authorization state is invalid")
	}
	if err := ValidateMCPCallbackURL(input.CallbackURL); err != nil {
		return err
	}
	return nil
}

// ValidateMCPCallbackURL permits only an explicit HTTP loopback callback. It
// prevents a browser login from sending an authorization code to a remote URL.
func ValidateMCPCallbackURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return apperror.New(apperror.KindValidation, "MCP callback URL must be an HTTP loopback URL")
	}
	host := parsed.Hostname()
	if host == "" || !isLoopbackHost(host) || parsed.Port() == "" || parsed.Path != "/mcp/callback" || parsed.RawQuery != "" {
		return apperror.New(apperror.KindValidation, "MCP callback URL must be an HTTP loopback URL")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func newMCPGrantCode() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate MCP authorization code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (s *mcpGrantStore) put(code string, value mcpGrantRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(time.Now().UTC())
	s.grants[sha256.Sum256([]byte(code))] = value
}

func (s *mcpGrantStore) consume(code string, now time.Time) (string, error) {
	key := sha256.Sum256([]byte(strings.TrimSpace(code)))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(now)
	value, ok := s.grants[key]
	if !ok || !now.Before(value.expiresAt) {
		delete(s.grants, key)
		return "", apperror.New(apperror.KindUnauthorized, "MCP authorization code is invalid or expired")
	}
	delete(s.grants, key)
	return value.userID, nil
}

func (s *mcpGrantStore) prune(now time.Time) {
	for key, value := range s.grants {
		if !now.Before(value.expiresAt) {
			delete(s.grants, key)
		}
	}
}
