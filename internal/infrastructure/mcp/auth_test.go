package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestTokenStorePersistsPrivateCredential(t *testing.T) {
	store := NewTokenStore(filepath.Join(t.TempDir(), "credentials", "mcp-token"))
	if err := store.Save("token-value"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	value, err := store.Load()
	if err != nil || value != "token-value" {
		t.Fatalf("Load() = %q, %v", value, err)
	}
	info, err := os.Stat(store.path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("credential mode = %o, want private", info.Mode().Perm())
	}
}

func TestBrowserAuthorizerDeliversLoopbackCodeAndExchangesIt(t *testing.T) {
	var exchangedCode string
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/auth/mcp-grant/exchange" {
			http.NotFound(writer, request)
			return
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != `{"code":"one-time-code"}` {
			t.Fatalf("exchange body = %s", body)
		}
		exchangedCode = "one-time-code"
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"access_token":"bearer-token"}`))
	}))
	defer api.Close()

	authorizer, err := NewBrowserAuthorizer(AuthorizerConfig{
		APIURL:  api.URL,
		WebURL:  "https://orbit.example",
		Timeout: time.Second,
		Browser: browserFunc(func(rawURL string) error {
			browserURL, err := url.Parse(rawURL)
			if err != nil {
				return err
			}
			callbackURL, err := url.Parse(browserURL.Query().Get("callback_url"))
			if err != nil {
				return err
			}
			callbackURL.RawQuery = url.Values{"code": {"one-time-code"}, "state": {browserURL.Query().Get("state")}}.Encode()
			response, err := (&http.Client{
				CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}).Get(callbackURL.String())
			if err != nil {
				return err
			}
			defer func() { _ = response.Body.Close() }()
			if response.StatusCode != http.StatusSeeOther {
				t.Fatalf("callback status = %d", response.StatusCode)
			}
			if location := response.Header.Get("Location"); location != "https://orbit.example/mcp/callback" {
				t.Fatalf("callback location = %q", location)
			}
			return nil
		}),
	})
	if err != nil {
		t.Fatalf("NewBrowserAuthorizer() error = %v", err)
	}
	token, err := authorizer.Authorize(context.Background())
	if err != nil || token != "bearer-token" || exchangedCode != "one-time-code" {
		t.Fatalf("Authorize() = %q, %v; exchanged = %q", token, err, exchangedCode)
	}
}

func TestBrowserAuthorizerCallbackRedirectsToFrontend(t *testing.T) {
	result := make(chan callbackResult, 1)
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:48123/mcp/callback?code=one-time-code&state=expected-state", nil)
	request.RemoteAddr = "127.0.0.1:48123"
	response := httptest.NewRecorder()

	BrowserAuthorizer{webURL: "https://orbit.example"}.callbackHandler("expected-state", result).ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("callback status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if location := response.Header().Get("Location"); location != "https://orbit.example/mcp/callback" {
		t.Fatalf("callback location = %q", location)
	}
	if callback := <-result; callback.code != "one-time-code" {
		t.Fatalf("callback code = %q", callback.code)
	}
}

func TestBrowserAuthorizerRejectedCallbackRedirectsToFrontend(t *testing.T) {
	result := make(chan callbackResult, 1)
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:48123/mcp/callback?code=one-time-code&state=unexpected-state", nil)
	request.RemoteAddr = "127.0.0.1:48123"
	response := httptest.NewRecorder()

	BrowserAuthorizer{webURL: "https://orbit.example"}.callbackHandler("expected-state", result).ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("callback status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	if location := response.Header().Get("Location"); location != "https://orbit.example/mcp/callback?status=error" {
		t.Fatalf("callback location = %q", location)
	}
	if callback := <-result; callback.err == nil {
		t.Fatal("callback error = nil")
	}
}

type browserFunc func(string) error

func (f browserFunc) Open(rawURL string) error {
	return f(rawURL)
}
