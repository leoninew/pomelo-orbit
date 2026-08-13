package csrf

import (
	"testing"

	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
)

const testKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestIssueAndVerify(t *testing.T) {
	token, err := Issue(testKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(testKey, token); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsWrongType(t *testing.T) {
	token, err := security.EncryptString(testKey, "credential")
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(testKey, token); err == nil {
		t.Fatal("expected type mismatch")
	}
}

func TestVerifyRejectsEmptyAndTampered(t *testing.T) {
	if err := Verify(testKey, ""); err == nil {
		t.Fatal("expected empty token error")
	}
	token, err := Issue(testKey)
	if err != nil {
		t.Fatal(err)
	}
	tampered := token[:len(token)-2] + "AA"
	if err := Verify(testKey, tampered); err == nil {
		t.Fatal("expected tampered token error")
	}
}
