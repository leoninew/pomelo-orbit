package authsvc

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	authport "github.com/leoninew/pomelo-orbit/internal/application/auth/port"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type googleProviderFacts struct {
	identity                     authport.GoogleIdentity
	err                          error
	state, nonce, verifier, code string
	calls                        int
	onExchange                   func()
}

func (p *googleProviderFacts) AuthorizationURL(state, nonce, verifier string) string {
	p.state, p.nonce, p.verifier = state, nonce, verifier
	return "https://accounts.google.com/authorize?state=" + state
}

func (p *googleProviderFacts) Exchange(_ context.Context, code, verifier, nonce string) (authport.GoogleIdentity, error) {
	p.calls++
	p.code = code
	if p.onExchange != nil {
		p.onExchange()
	}
	if verifier != p.verifier || nonce != p.nonce {
		return authport.GoogleIdentity{}, errors.New("challenge does not match authorization")
	}
	return p.identity, p.err
}

type googleUserFacts struct {
	repository.UserStore
	repository.AuthStore
	subjectUser, emailUser model.User
	subjectErr, emailErr   error
	created, updated       *model.User
	loggedIn               string
	history                []model.LoginHistory
}

func (f *googleUserFacts) UserByOAuthAccount(context.Context, string, string) (model.User, error) {
	return f.subjectUser, f.subjectErr
}
func (f *googleUserFacts) UserByEmail(context.Context, string) (model.User, error) {
	return f.emailUser, f.emailErr
}
func (f *googleUserFacts) UserByUsername(context.Context, string) (model.User, error) {
	return model.User{}, repository.ErrNotFound
}
func (f *googleUserFacts) CreateUser(_ context.Context, user model.User) error {
	f.created = &user
	return nil
}
func (f *googleUserFacts) UpdateUser(_ context.Context, user model.User) error {
	f.updated = &user
	return nil
}
func (f *googleUserFacts) MarkUserLoggedIn(_ context.Context, userId string) error {
	f.loggedIn = userId
	return nil
}
func (f *googleUserFacts) SaveLoginHistory(_ context.Context, history model.LoginHistory) error {
	f.history = append(f.history, history)
	return nil
}

type googleTransaction func(context.Context, func(context.Context) error) error

func (run googleTransaction) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return run(ctx, fn)
}

func newGoogleTestService(facts *googleUserFacts, provider *googleProviderFacts, run googleTransaction) GoogleService {
	auth := New(facts, facts, jwt.NewTokenService(authTestSecretKey), nil, authTestSecretKey)
	return NewGoogle(auth, provider, run)
}

func completeGoogle(t *testing.T, service GoogleService, provider *googleProviderFacts) (authdto.GoogleLoginResult, error) {
	t.Helper()
	login, err := service.Begin("/projects?tab=mine")
	if err != nil {
		t.Fatal(err)
	}
	return service.Complete(context.Background(), authdto.GoogleCallbackInput{Code: "google-code", State: provider.state, Session: login.Session, IP: "127.0.0.1", UserAgent: "test"})
}

func TestGoogleLoginResolvesAccounts(t *testing.T) {
	for _, scenario := range []string{"subject", "email", "new"} {
		t.Run(scenario, func(t *testing.T) {
			facts := &googleUserFacts{subjectErr: repository.ErrNotFound, emailErr: repository.ErrNotFound}
			existing := model.User{Id: "existing-user", Username: "existing", Status: "enabled", Email: "user@example.test", PasswordHash: "preserved-password"}
			switch scenario {
			case "subject":
				facts.subjectUser, facts.subjectErr = existing, nil
			case "email":
				facts.emailUser, facts.emailErr = existing, nil
			}
			inTransaction := false
			provider := &googleProviderFacts{identity: authport.GoogleIdentity{Subject: "google-subject", Email: "USER@example.test", EmailVerified: true, Name: "Google User"}, onExchange: func() {
				if inTransaction {
					t.Fatal("Google exchange must precede the transaction")
				}
			}}
			service := newGoogleTestService(facts, provider, func(ctx context.Context, fn func(context.Context) error) error {
				inTransaction = true
				defer func() { inTransaction = false }()
				return fn(ctx)
			})
			result, err := completeGoogle(t, service, provider)
			if err != nil {
				t.Fatal(err)
			}
			claims, err := service.auth.tokens.Verify(result.Token)
			if err != nil || claims.Sub != facts.loggedIn || result.Redirect != "/projects?tab=mine" || provider.code != "google-code" {
				t.Fatalf("unexpected login result=%+v claims=%+v error=%v", result, claims, err)
			}
			if len(facts.history) != 1 || facts.history[0].UserId != claims.Sub || !facts.history[0].Success {
				t.Fatalf("login history=%+v", facts.history)
			}
			switch scenario {
			case "subject":
				if claims.Sub != existing.Id || facts.updated != nil || facts.created != nil {
					t.Fatal("subject login must reuse the existing account")
				}
			case "email":
				if facts.updated == nil || facts.updated.OAuthProviderId != "google-subject" || facts.updated.PasswordHash != existing.PasswordHash || facts.created != nil {
					t.Fatal("email login must preserve and link the existing account")
				}
			case "new":
				if facts.created == nil || facts.created.Email != "user@example.test" || facts.created.PasswordHash != "" || facts.created.AuthSource != "oauth" {
					t.Fatal("new Google account was not persisted correctly")
				}
			}
		})
	}
}

func TestGoogleLoginRejectsUnsafeIdentities(t *testing.T) {
	for _, scenario := range []string{"disabled subject", "disabled email", "linked email", "unverified email", "repository failure", "upstream failure"} {
		t.Run(scenario, func(t *testing.T) {
			facts := &googleUserFacts{subjectErr: repository.ErrNotFound, emailErr: repository.ErrNotFound}
			provider := &googleProviderFacts{identity: authport.GoogleIdentity{Subject: "subject", Email: "user@example.test", EmailVerified: true}}
			var want apperror.Kind
			switch scenario {
			case "disabled subject":
				facts.subjectUser, facts.subjectErr = model.User{Status: "disabled"}, nil
				want = apperror.KindForbidden
			case "disabled email":
				facts.emailUser, facts.emailErr = model.User{Status: "disabled"}, nil
				want = apperror.KindForbidden
			case "linked email":
				facts.emailUser, facts.emailErr = model.User{Status: "enabled", OAuthProvider: "google", OAuthProviderId: "different-subject"}, nil
				want = apperror.KindConflict
			case "unverified email":
				provider.identity.EmailVerified = false
				want = apperror.KindValidation
			case "repository failure":
				facts.subjectErr = errors.New("database unavailable")
			case "upstream failure":
				provider.err = apperror.New(apperror.KindUnavailable, "Google is unavailable")
				want = apperror.KindUnavailable
			}
			service := newGoogleTestService(facts, provider, func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			result, err := completeGoogle(t, service, provider)
			if err == nil || result.Token != "" || (want != "" && !apperror.IsKind(err, want)) || facts.created != nil || facts.updated != nil || facts.loggedIn != "" {
				t.Fatalf("unsafe identity accepted: result=%+v error=%v", result, err)
			}
		})
	}
}

func TestGoogleSessionBindingAndExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		provider := &googleProviderFacts{}
		service := newGoogleTestService(nil, provider, nil)
		login, err := service.Begin("/")
		if err != nil {
			t.Fatal(err)
		}
		input := authdto.GoogleCallbackInput{Code: "code", State: provider.state, Session: login.Session}
		for _, invalid := range []authdto.GoogleCallbackInput{
			{Code: input.Code, State: "wrong-state", Session: input.Session},
			{Code: input.Code, State: input.State},
			{Code: input.Code, State: input.State, Session: input.Session + "tampered"},
		} {
			if _, err := service.Complete(context.Background(), invalid); !apperror.IsKind(err, apperror.KindValidation) {
				t.Fatalf("invalid session error=%v", err)
			}
		}
		time.Sleep(GoogleSessionTtl + time.Second)
		if _, err := service.Complete(context.Background(), input); !apperror.IsKind(err, apperror.KindValidation) {
			t.Fatalf("expired session error=%v", err)
		}
		if provider.calls != 0 {
			t.Fatal("invalid sessions reached Google")
		}
	})
}

func TestGoogleLoginDoesNotReturnTokenOnCommitFailure(t *testing.T) {
	facts := &googleUserFacts{subjectUser: model.User{Id: "user", Username: "user", Status: "enabled"}}
	provider := &googleProviderFacts{identity: authport.GoogleIdentity{Subject: "subject", Email: "user@example.test", EmailVerified: true}}
	commitError := errors.New("commit failed")
	service := newGoogleTestService(facts, provider, func(ctx context.Context, fn func(context.Context) error) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return commitError
	})
	result, err := completeGoogle(t, service, provider)
	if !errors.Is(err, commitError) || result.Token != "" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestGoogleLoginRejectsExternalRedirects(t *testing.T) {
	service := newGoogleTestService(nil, &googleProviderFacts{}, nil)
	for _, redirect := range []string{"https://example.com", "//example.com", "/\\example.com"} {
		if _, err := service.Begin(redirect); !apperror.IsKind(err, apperror.KindValidation) {
			t.Fatalf("redirect=%q error=%v", redirect, err)
		}
	}
}
