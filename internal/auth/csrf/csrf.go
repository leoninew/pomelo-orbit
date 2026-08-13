package csrf

import (
	"errors"
	"strings"
	"time"

	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
)

const (
	// TokenTTL is the cryptographic lifetime of a login CSRF token.
	TokenTTL = 3 * time.Minute
	// payloadType is the Fernet plaintext type marker. It prevents credential
	// ciphertexts and other Fernet uses from being accepted as CSRF tokens.
	payloadType = "csrf"
)

// Issue creates a Fernet-encrypted CSRF token using the configured secret key.
func Issue(secretKey string) (string, error) {
	return security.EncryptString(secretKey, payloadType)
}

// Verify checks that token is a valid, unexpired CSRF Fernet token.
func Verify(secretKey string, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("csrf token is empty")
	}
	plain, err := security.DecryptStringWithTTL(secretKey, token, TokenTTL)
	if err != nil {
		return err
	}
	if plain != payloadType {
		return errors.New("csrf token type mismatch")
	}
	return nil
}
