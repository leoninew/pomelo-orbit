package port

import "context"

type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

// GoogleProvider exchanges a browser authorization for a verified Google identity.
type GoogleProvider interface {
	AuthorizationURL(state, nonce, verifier string) string
	Exchange(ctx context.Context, code, verifier, nonce string) (GoogleIdentity, error)
}

// TransactionRunner owns the short persistence phase after Google verification.
type TransactionRunner interface {
	RunInTransaction(context.Context, func(context.Context) error) error
}
