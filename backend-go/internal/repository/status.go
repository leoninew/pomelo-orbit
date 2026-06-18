package repository

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

func NewId() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return strings.ToLower(encoded)[:26]
}
