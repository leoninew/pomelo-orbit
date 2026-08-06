// Package mcp provides local-only infrastructure for the Go stdio MCP entry.
package mcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
)

const tokenFileName = "mcp-token"

type TokenStore struct {
	path string
}

func NewDefaultTokenStore() (TokenStore, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return TokenStore{}, fmt.Errorf("locate user config directory: %w", err)
	}
	return TokenStore{path: filepath.Join(directory, "PomeloOrbit", tokenFileName)}, nil
}

func NewTokenStore(path string) TokenStore {
	return TokenStore{path: path}
}

func (s TokenStore) Load() (string, error) {
	contents, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read MCP credential store: %w", err)
	}
	return strings.TrimSpace(string(contents)), nil
}

func (s TokenStore) Save(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("MCP credential token is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create MCP credential directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), tokenFileName+"-*")
	if err != nil {
		return fmt.Errorf("create MCP credential file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect MCP credential file: %w", err)
	}
	if _, err := temporary.WriteString(token + "\n"); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write MCP credential file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close MCP credential file: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("install MCP credential file: %w", err)
	}
	return nil
}

type BrowserLauncher interface {
	Open(string) error
}

type SystemBrowserLauncher struct{}

func (SystemBrowserLauncher) Open(rawURL string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	case "darwin":
		command = exec.Command("open", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("open browser for MCP authorization: %w", err)
	}
	return nil
}

type AuthorizerConfig struct {
	APIURL     string
	WebURL     string
	Timeout    time.Duration
	HTTPClient *http.Client
	Browser    BrowserLauncher
}

type BrowserAuthorizer struct {
	apiURL     string
	webURL     string
	timeout    time.Duration
	httpClient *http.Client
	browser    BrowserLauncher
}

func NewBrowserAuthorizer(cfg AuthorizerConfig) (BrowserAuthorizer, error) {
	apiURL := strings.TrimRight(strings.TrimSpace(cfg.APIURL), "/")
	webURL := strings.TrimRight(strings.TrimSpace(cfg.WebURL), "/")
	if _, err := url.ParseRequestURI(apiURL); err != nil {
		return BrowserAuthorizer{}, fmt.Errorf("invalid MCP API URL: %w", err)
	}
	if _, err := url.ParseRequestURI(webURL); err != nil {
		return BrowserAuthorizer{}, fmt.Errorf("invalid MCP web URL: %w", err)
	}
	if cfg.Timeout <= 0 {
		return BrowserAuthorizer{}, errors.New("MCP browser authorization timeout must be positive")
	}
	if cfg.Browser == nil {
		cfg.Browser = SystemBrowserLauncher{}
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return BrowserAuthorizer{apiURL: apiURL, webURL: webURL, timeout: cfg.Timeout, httpClient: cfg.HTTPClient, browser: cfg.Browser}, nil
}

func (a BrowserAuthorizer) Authorize(ctx context.Context) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("start MCP authorization callback: %w", err)
	}
	defer func() { _ = listener.Close() }()

	state, err := randomState()
	if err != nil {
		return "", err
	}
	callbackURL := "http://" + listener.Addr().String() + "/mcp/callback"
	if err := authsvc.ValidateMCPCallbackURL(callbackURL); err != nil {
		return "", fmt.Errorf("construct MCP authorization callback: %w", err)
	}
	callback := make(chan callbackResult, 1)
	server := &http.Server{Handler: a.callbackHandler(state, callback)}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Close() }()

	authorizationURL := a.webURL + "/mcp-authorize?" + url.Values{
		"callback_url": {callbackURL},
		"state":        {state},
	}.Encode()
	if err := a.browser.Open(authorizationURL); err != nil {
		return "", err
	}

	waitCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	select {
	case result := <-callback:
		if result.err != nil {
			return "", result.err
		}
		return a.exchange(waitCtx, result.code)
	case <-waitCtx.Done():
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return "", errors.New("MCP browser authorization timed out")
		}
		return "", waitCtx.Err()
	}
}

type callbackResult struct {
	code string
	err  error
}

func (a BrowserAuthorizer) callbackHandler(state string, result chan<- callbackResult) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/mcp/callback" || !isLoopbackRemoteAddress(request.RemoteAddr) {
			http.NotFound(writer, request)
			return
		}
		code := strings.TrimSpace(request.URL.Query().Get("code"))
		returnedState := request.URL.Query().Get("state")
		if code == "" || subtle.ConstantTimeCompare([]byte(returnedState), []byte(state)) != 1 {
			http.Error(writer, "MCP authorization callback was rejected", http.StatusBadRequest)
			select {
			case result <- callbackResult{err: errors.New("MCP authorization callback state did not match")}:
			default:
			}
			return
		}
		writer.WriteHeader(http.StatusNoContent)
		select {
		case result <- callbackResult{code: code}:
		default:
		}
	})
}

func (a BrowserAuthorizer) exchange(ctx context.Context, code string) (string, error) {
	body, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL+"/api/auth/mcp-grant/exchange", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create MCP authorization exchange request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := a.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("exchange MCP authorization code: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("exchange MCP authorization code: server returned %s", response.Status)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode MCP authorization exchange response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", errors.New("MCP authorization exchange returned no token")
	}
	return payload.AccessToken, nil
}

func randomState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate MCP authorization state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func isLoopbackRemoteAddress(remoteAddress string) bool {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
