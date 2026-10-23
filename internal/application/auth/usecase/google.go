package authsvc

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	authport "github.com/leoninew/pomelo-orbit/internal/application/auth/port"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const GoogleSessionTtl = 10 * time.Minute

type GoogleService struct {
	auth         Service
	provider     authport.GoogleProvider
	transactions authport.TransactionRunner
}

func NewGoogle(auth Service, provider authport.GoogleProvider, transactions authport.TransactionRunner) GoogleService {
	return GoogleService{auth: auth, provider: provider, transactions: transactions}
}

type googleSession struct {
	Type     string `json:"type"`
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
	Redirect string `json:"redirect"`
}

func (s GoogleService) Begin(redirect string) (authdto.GoogleAuthorization, error) {
	if s.provider == nil {
		return authdto.GoogleAuthorization{}, apperror.New(apperror.KindUnavailable, "Google OAuth is not configured")
	}
	target, err := googleRedirect(redirect)
	if err != nil {
		return authdto.GoogleAuthorization{}, err
	}
	session := googleSession{Type: "google-login", State: googleRandom(), Nonce: googleRandom(), Verifier: googleRandom(), Redirect: target}
	payload, err := json.Marshal(session)
	if err != nil {
		return authdto.GoogleAuthorization{}, err
	}
	encrypted, err := security.EncryptString(s.auth.secretKey, string(payload))
	if err != nil {
		return authdto.GoogleAuthorization{}, err
	}
	return authdto.GoogleAuthorization{URL: s.provider.AuthorizationURL(session.State, session.Nonce, session.Verifier), Session: encrypted}, nil
}

func (s GoogleService) Complete(ctx context.Context, input authdto.GoogleCallbackInput) (authdto.GoogleLoginResult, error) {
	if s.provider == nil {
		return authdto.GoogleLoginResult{}, apperror.New(apperror.KindUnavailable, "Google OAuth is not configured")
	}
	var session googleSession
	payload, err := security.DecryptStringWithTtl(s.auth.secretKey, input.Session, GoogleSessionTtl)
	if err != nil || json.Unmarshal([]byte(payload), &session) != nil || session.Type != "google-login" || session.State == "" || input.State == "" || session.Nonce == "" || session.Verifier == "" || input.Code == "" || subtle.ConstantTimeCompare([]byte(session.State), []byte(input.State)) != 1 {
		return authdto.GoogleLoginResult{}, apperror.New(apperror.KindValidation, "Google login session is invalid or expired; please sign in again")
	}
	identity, err := s.provider.Exchange(ctx, input.Code, session.Verifier, session.Nonce)
	if err != nil {
		return authdto.GoogleLoginResult{}, err
	}
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	address, emailErr := mail.ParseAddress(email)
	if identity.Subject == "" || !identity.EmailVerified || emailErr != nil || address.Address != email {
		return authdto.GoogleLoginResult{}, apperror.New(apperror.KindValidation, "Google account must provide a verified email address")
	}
	identity.Email = email
	var user model.User
	err = s.transactions.RunInTransaction(ctx, func(ctx context.Context) error {
		var err error
		user, err = s.resolveUser(ctx, identity)
		if err != nil {
			return err
		}
		if err := s.auth.repo.MarkUserLoggedIn(ctx, user.Id); err != nil {
			return err
		}
		return s.auth.auth.SaveLoginHistory(ctx, model.LoginHistory{Id: idutil.NewId(), UserId: user.Id, Username: user.Username, IpAddress: optionalString(input.IP), UserAgent: optionalString(input.UserAgent), LoginAt: time.Now().UTC(), Success: true})
	})
	if err != nil {
		return authdto.GoogleLoginResult{}, err
	}
	token, err := s.auth.tokens.Sign(user.Id, user.Username)
	if err != nil {
		return authdto.GoogleLoginResult{}, err
	}
	return authdto.GoogleLoginResult{Token: token, Redirect: session.Redirect}, nil
}

func (s GoogleService) resolveUser(ctx context.Context, identity authport.GoogleIdentity) (model.User, error) {
	user, err := s.auth.repo.UserByOAuthAccount(ctx, "google", identity.Subject)
	if err == nil {
		return enabledGoogleUser(user)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return model.User{}, err
	}
	user, err = s.auth.repo.UserByEmail(ctx, identity.Email)
	if err == nil {
		if _, err := enabledGoogleUser(user); err != nil {
			return model.User{}, err
		}
		if user.OAuthProvider != "" || user.OAuthProviderId != "" {
			return model.User{}, apperror.New(apperror.KindConflict, "This email address is already linked to another sign-in identity")
		}
		user.OAuthProvider, user.OAuthProviderId, user.AuthSource = "google", identity.Subject, "oauth"
		return user, s.auth.repo.UpdateUser(ctx, user)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return model.User{}, err
	}
	username, err := s.uniqueGoogleUsername(ctx, identity)
	if err != nil {
		return model.User{}, err
	}
	now := time.Now().UTC()
	user = model.User{Id: idutil.NewId(), Username: username, Status: "enabled", OAuthProvider: "google", OAuthProviderId: identity.Subject, Email: identity.Email, AuthSource: "oauth", CreatedAt: now, UpdatedAt: now}
	return user, s.auth.repo.CreateUser(ctx, user)
}

func (s GoogleService) uniqueGoogleUsername(ctx context.Context, identity authport.GoogleIdentity) (string, error) {
	base := strings.TrimSpace(identity.Name)
	if base == "" {
		base, _, _ = strings.Cut(identity.Email, "@")
	}
	for len(base) > 40 {
		_, size := utf8.DecodeLastRuneInString(base)
		base = base[:len(base)-size]
	}
	for suffix := 0; suffix < 100; suffix++ {
		username := base
		if suffix > 0 {
			username += strconv.Itoa(suffix)
		}
		if _, err := s.auth.repo.UserByUsername(ctx, username); errors.Is(err, repository.ErrNotFound) {
			return username, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", apperror.New(apperror.KindConflict, "Unable to allocate a username for this Google account")
}

func enabledGoogleUser(user model.User) (model.User, error) {
	if user.Status != "enabled" {
		return model.User{}, apperror.New(apperror.KindForbidden, "This account is disabled")
	}
	return user, nil
}

func googleRandom() string {
	value := make([]byte, 32)
	_, _ = rand.Read(value) // crypto/rand.Read always fills the slice or terminates the process.
	return base64.RawURLEncoding.EncodeToString(value)
}

func googleRedirect(value string) (string, error) {
	if value == "" {
		return "/", nil
	}
	target, err := url.Parse(value)
	if err != nil || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\\\r\n") || target.IsAbs() || target.Host != "" || len(value) > 2048 {
		return "", apperror.New(apperror.KindValidation, "Google login redirect must be a local path")
	}
	return value, nil
}
