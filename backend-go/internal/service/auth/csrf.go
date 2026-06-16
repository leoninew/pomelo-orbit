package authsvc

import (
	"crypto/rand"
	"encoding/hex"
)

func (s Service) NewCSRFToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
