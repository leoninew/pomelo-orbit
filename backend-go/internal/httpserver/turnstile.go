package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend/internal/config"
)

type turnstileVerifier interface {
	Verify(ctx context.Context, token string, remoteIP string) error
}

type cloudflareTurnstileVerifier struct {
	secretKey string
	verifyURL string
	client    *http.Client
}

type turnstileVerifyReq struct {
	Secret   string `json:"secret"`
	Response string `json:"response"`
	RemoteIP string `json:"remoteip,omitempty"`
}

type turnstileVerifyResp struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Hostname   string   `json:"hostname"`
	Action     string   `json:"action"`
}

func newTurnstileVerifier(cfg config.TurnstileConfig) turnstileVerifier {
	return cloudflareTurnstileVerifier{
		secretKey: cfg.SecretKey,
		verifyURL: cfg.VerifyURL,
		client:    &http.Client{Timeout: 3 * time.Second},
	}
}

func (v cloudflareTurnstileVerifier) Verify(ctx context.Context, token string, remoteIP string) error {
	payload, err := json.Marshal(turnstileVerifyReq{Secret: v.secretKey, Response: token, RemoteIP: remoteIP})
	if err != nil {
		return fmt.Errorf("marshal turnstile verification request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.verifyURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create turnstile verification request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("verify turnstile token: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("turnstile verification failed: status=%d", response.StatusCode)
	}

	var result turnstileVerifyResp
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode turnstile verification response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("turnstile verification rejected: error_codes=%v", result.ErrorCodes)
	}
	return nil
}
