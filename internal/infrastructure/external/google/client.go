package google

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	authport "github.com/leoninew/pomelo-orbit/internal/application/auth/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"golang.org/x/oauth2"
)

const googleIssuer = "https://accounts.google.com"

type Client struct {
	oauth    oauth2.Config
	http     *http.Client
	verifier *oidc.IDTokenVerifier
}

func New(cfg config.GoogleConfig) *Client {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	ctx := oidc.ClientContext(context.Background(), httpClient)
	return &Client{
		oauth:    oauth2.Config{ClientID: cfg.ClientId, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectUri, Scopes: []string{oidc.ScopeOpenID, "email", "profile"}, Endpoint: oauth2.Endpoint{AuthURL: googleIssuer + "/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token", AuthStyle: oauth2.AuthStyleInParams}},
		http:     httpClient,
		verifier: oidc.NewVerifier(googleIssuer, oidc.NewRemoteKeySet(ctx, "https://www.googleapis.com/oauth2/v3/certs"), &oidc.Config{ClientID: cfg.ClientId, SupportedSigningAlgs: []string{oidc.RS256}}),
	}
}

func (c *Client) AuthorizationURL(state, nonce, verifier string) string {
	return c.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("nonce", nonce), oauth2.SetAuthURLParam("prompt", "select_account"))
}

func (c *Client) Exchange(ctx context.Context, code, verifier, nonce string) (authport.GoogleIdentity, error) {
	ctx = oidc.ClientContext(ctx, c.http)
	token, err := c.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		var response *oauth2.RetrieveError
		if errors.As(err, &response) {
			// RetrieveError includes the raw response body; never propagate it to logs.
			if response.ErrorCode == "invalid_grant" {
				return authport.GoogleIdentity{}, apperror.New(apperror.KindValidation, "Google authorization is invalid or expired; please sign in again")
			}
			return authport.GoogleIdentity{}, apperror.Wrap(apperror.KindUnavailable, "Google authorization is unavailable; please try again", fmt.Errorf("token endpoint returned status %d", response.Response.StatusCode))
		}
		return authport.GoogleIdentity{}, apperror.Wrap(apperror.KindUnavailable, "Google authorization request failed; please try again", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return authport.GoogleIdentity{}, apperror.New(apperror.KindValidation, "Google did not return an identity token")
	}
	verified, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return authport.GoogleIdentity{}, apperror.Wrap(apperror.KindValidation, "Google identity verification failed; please sign in again", err)
	}
	if nonce == "" || subtle.ConstantTimeCompare([]byte(verified.Nonce), []byte(nonce)) != 1 {
		return authport.GoogleIdentity{}, apperror.New(apperror.KindValidation, "Google login nonce is invalid")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := verified.Claims(&claims); err != nil {
		return authport.GoogleIdentity{}, apperror.New(apperror.KindValidation, "Google identity claims are invalid")
	}
	return authport.GoogleIdentity{Subject: verified.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified, Name: claims.Name}, nil
}
