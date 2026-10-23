package google

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestGoogleAuthorizationUsesPKCEAndNonce(t *testing.T) {
	client := New(config.GoogleConfig{ClientId: "client", ClientSecret: "secret", RedirectUri: "http://localhost:9020/google/callback"})
	authorization, err := url.Parse(client.AuthorizationURL("state-value", "nonce-value", "verifier-value"))
	if err != nil {
		t.Fatal(err)
	}
	query := authorization.Query()
	challenge := sha256.Sum256([]byte("verifier-value"))
	if authorization.Host != "accounts.google.com" || query.Get("state") != "state-value" || query.Get("nonce") != "nonce-value" || query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(challenge[:]) || strings.Contains(authorization.String(), "secret") {
		t.Fatalf("unexpected authorization URL: %s", authorization)
	}
}

func TestGoogleExchangeVerifiesSignedIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"valid", "issuer", "audience", "expired", "nonce", "signature", "invalid grant", "upstream", "missing token"} {
		t.Run(scenario, func(t *testing.T) {
			claims := map[string]any{"iss": googleIssuer, "aud": "client", "sub": "google-subject", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": "nonce", "email": "user@example.test", "email_verified": true, "name": "Google User"}
			switch scenario {
			case "issuer":
				claims["iss"] = "https://attacker.example"
			case "audience":
				claims["aud"] = "other-client"
			case "expired":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "nonce":
				claims["nonce"] = "other-nonce"
			}
			token := signedGoogleToken(t, key, claims)
			if scenario == "signature" {
				parts := strings.Split(token, ".")
				parts[2] = base64.RawURLEncoding.EncodeToString(make([]byte, 256))
				token = strings.Join(parts, ".")
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/keys" {
					_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "test-key", "alg": "RS256", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
					return
				}
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				if r.Form.Get("code") != "authorization-code" || r.Form.Get("code_verifier") != "pkce-verifier" || r.Form.Get("client_secret") != "client-secret" || r.Form.Get("redirect_uri") != "http://localhost:9020/google/callback" {
					t.Error("authorization exchange lost its code, verifier or configuration")
				}
				switch scenario {
				case "invalid grant":
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"sensitive-upstream-body"}`))
				case "upstream":
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"error":"sensitive-upstream-body"}`))
				case "missing token":
					_, _ = w.Write([]byte(`{"access_token":"google-access-token","token_type":"Bearer"}`))
				default:
					_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "google-access-token", "token_type": "Bearer", "id_token": token})
				}
			}))
			defer server.Close()
			client := New(config.GoogleConfig{ClientId: "client", ClientSecret: "client-secret", RedirectUri: "http://localhost:9020/google/callback"})
			client.oauth.Endpoint.TokenURL = server.URL + "/token"
			client.verifier = oidc.NewVerifier(googleIssuer, oidc.NewRemoteKeySet(context.Background(), server.URL+"/keys"), &oidc.Config{ClientID: "client", SupportedSigningAlgs: []string{oidc.RS256}})
			identity, err := client.Exchange(context.Background(), "authorization-code", "pkce-verifier", "nonce")
			if scenario == "valid" {
				if err != nil || identity.Subject != "google-subject" || !identity.EmailVerified || identity.Email != "user@example.test" {
					t.Fatalf("identity=%+v error=%v", identity, err)
				}
				return
			}
			want := apperror.KindValidation
			if scenario == "upstream" {
				want = apperror.KindUnavailable
			}
			if !apperror.IsKind(err, want) {
				t.Fatalf("error=%v want=%s", err, want)
			}
			if strings.Contains(err.Error(), "sensitive-upstream-body") || strings.Contains(err.Error(), "google-access-token") {
				t.Fatal("Google credentials leaked into the propagated error")
			}
		})
	}
}

func signedGoogleToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"test-key"}`))
	signed := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signed))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature)
}
