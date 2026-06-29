package security

import "testing"

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

func TestFernetDerivesKeyFromPlainSecret(t *testing.T) {
	secretKey := "plain-jwt-secret-with-at-least-32-chars"
	encrypted, err := EncryptString(secretKey, "secret")
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := DecryptString(secretKey, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != "secret" {
		t.Fatalf("unexpected decrypted value: %q", decrypted)
	}
}

func TestFernetRejectsShortPlainSecret(t *testing.T) {
	if _, err := EncryptString("short-secret", "secret"); err == nil {
		t.Fatal("expected short secret error")
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
