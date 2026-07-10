package turnstile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func TestVerifierSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
		var req verifyReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Secret != "secret" || req.Response != "token" || req.RemoteIP != "127.0.0.1" {
			t.Fatalf("unexpected request: %+v", req)
		}
		writeJSON(w, http.StatusOK, verifyResp{Success: true})
	}))
	defer server.Close()

	verifier := NewVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
}

func TestVerifierRejectsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, verifyResp{Success: false, ErrorCodes: []string{"invalid-input-response"}})
	}))
	defer server.Close()

	verifier := NewVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifierRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	verifier := NewVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifierRejectsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	verifier := NewVerifier(config.TurnstileConfig{SecretKey: "secret", VerifyURL: server.URL})
	if err := verifier.Verify(t.Context(), "token", "127.0.0.1"); err == nil {
		t.Fatal("expected error")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
