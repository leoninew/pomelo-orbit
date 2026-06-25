package transporthttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/config"
	transportresponse "backend/internal/transport/http/response"
)

func TestTurnstileVerifierSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
		var req turnstileVerifyReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Secret != "secret" || req.Response != "token" || req.RemoteIP != "127.0.0.1" {
			t.Fatalf("unexpected request: %+v", req)
		}
		transportresponse.JSON(slog.Default(), w, http.StatusOK, turnstileVerifyResp{Success: true})
	}))
	defer server.Close()

	verifier := newTurnstileVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
}

func TestTurnstileVerifierRejectsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		transportresponse.JSON(slog.Default(), w, http.StatusOK, turnstileVerifyResp{Success: false, ErrorCodes: []string{"invalid-input-response"}})
	}))
	defer server.Close()

	verifier := newTurnstileVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestTurnstileVerifierRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	verifier := newTurnstileVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestTurnstileVerifierRejectsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	verifier := newTurnstileVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err == nil {
		t.Fatal("expected error")
	}
}
