package security

import (
	"crypto/rand"
	"io"
	"strings"
	"testing"
	"time"
)

const testFernetKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestFernetRoundTrip(t *testing.T) {
	plain := "-----BEGIN OPENSSH PRIVATE KEY-----\ncredential-data\n-----END OPENSSH PRIVATE KEY-----\n"
	encrypted, err := EncryptString(testFernetKey, plain)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == plain {
		t.Fatal("expected encrypted value to differ from plaintext")
	}
	decrypted, err := DecryptString(testFernetKey, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != plain {
		t.Fatalf("unexpected decrypted value: %q", decrypted)
	}
}

func TestFernetDecryptsPythonToken(t *testing.T) {
	decrypted, err := DecryptString(testFernetKey, "gAAAAABqL5rGh6HKnarq88jxOU1OrtgAbPNhO1Nou6se8sBpEEZaLthcXas2P22i5XrIu6ZCRc77o2nhb2gldVGrwaMrqCeFlcKK28VHhNx2m2B1bPZRimo=")
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "python-fernet-fixture" {
		t.Fatalf("unexpected decrypted fixture: %q", decrypted)
	}
}

func TestFernetRejectsInvalidKey(t *testing.T) {
	if _, err := EncryptString("not-a-fernet-key", "secret"); err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestFernetRejectsTamperedToken(t *testing.T) {
	encrypted, err := EncryptString(testFernetKey, "secret")
	if err != nil {
		t.Fatal(err)
	}
	tampered := encrypted[:len(encrypted)-2] + "AA"
	if _, err := DecryptString(testFernetKey, tampered); err == nil {
		t.Fatal("expected tampered token error")
	}
}

func TestFernetDecryptStringWithTTLAcceptsFreshToken(t *testing.T) {
	encrypted, err := EncryptString(testFernetKey, "csrf")
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := DecryptStringWithTTL(testFernetKey, encrypted, 3*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "csrf" {
		t.Fatalf("unexpected plaintext: %q", decrypted)
	}
}

func TestFernetDecryptStringWithTTLRejectsExpiredToken(t *testing.T) {
	key, err := parseFernetKey(testFernetKey)
	if err != nil {
		t.Fatal(err)
	}
	iv := make([]byte, fernetIVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-4 * time.Minute).Unix()
	encrypted, err := encryptFernet(key, []byte("csrf"), iv, past)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptStringWithTTL(testFernetKey, encrypted, 3*time.Minute); err == nil {
		t.Fatal("expected expired token error")
	} else if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}
	// Without TTL the ciphertext remains readable for credential-style decrypts.
	if plain, err := DecryptString(testFernetKey, encrypted); err != nil || plain != "csrf" {
		t.Fatalf("DecryptString should ignore TTL, got %q %v", plain, err)
	}
}

func TestFernetDecryptStringWithTTLRejectsTamperedToken(t *testing.T) {
	encrypted, err := EncryptString(testFernetKey, "csrf")
	if err != nil {
		t.Fatal(err)
	}
	tampered := encrypted[:len(encrypted)-2] + "AA"
	if _, err := DecryptStringWithTTL(testFernetKey, tampered, 3*time.Minute); err == nil {
		t.Fatal("expected tampered token error")
	}
}
