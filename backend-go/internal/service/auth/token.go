package authsvc

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
}

type TokenService struct {
	secret string
}

func NewTokenService(secret string) TokenService {
	return TokenService{secret: secret}
}

func (s TokenService) Sign(userId string, username string) (string, error) {
	claims, err := json.Marshal(Claims{Sub: userId, Username: username, Exp: time.Now().Add(24 * time.Hour).Unix()})
	if err != nil {
		return "", err
	}
	return s.signJWT(claims), nil
}

func (s TokenService) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token format")
	}
	signed := parts[0] + "." + parts[1]
	expected := s.jwtSignature(signed)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, err
	}
	if claims.Sub == "" || claims.Exp < time.Now().Unix() {
		return Claims{}, errors.New("token expired")
	}
	return claims, nil
}

func (s TokenService) signJWT(payload []byte) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	signed := header + "." + body
	return signed + "." + s.jwtSignature(signed)
}

func (s TokenService) jwtSignature(signed string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(signed))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
