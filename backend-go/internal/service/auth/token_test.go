package authsvc

import "testing"

func TestTokenServiceSignsAndVerifiesToken(t *testing.T) {
	service := NewTokenService("test-secret")
	token, err := service.Sign("user-1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sub != "user-1" || claims.Username != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenServiceRejectsInvalidSignature(t *testing.T) {
	service := NewTokenService("test-secret")
	token, err := service.Sign("user-1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	other := NewTokenService("other-secret")
	if _, err := other.Verify(token); err == nil {
		t.Fatal("expected invalid signature")
	}
}

func TestTokenServiceRejectsInvalidFormat(t *testing.T) {
	service := NewTokenService("test-secret")
	if _, err := service.Verify("not-a-token"); err == nil {
		t.Fatal("expected invalid token format")
	}
}
