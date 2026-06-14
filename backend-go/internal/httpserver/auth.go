package httpserver

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"backend/internal/orbit"

	"golang.org/x/crypto/bcrypt"
)

type loginReq struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	CSRFToken      string `json:"csrf_token"`
	TurnstileToken string `json:"turnstile_token"`
}

type passwordChangeReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type googleCallbackReq struct {
	Code string `json:"code"`
}

type loginHistoryResp struct {
	Id        string  `json:"id"`
	UserId    string  `json:"user_id"`
	Username  string  `json:"username"`
	IpAddress *string `json:"ip_address"`
	UserAgent *string `json:"user_agent"`
	LoginAt   string  `json:"login_at"`
	Success   bool    `json:"success"`
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type turnstileConfigResp struct {
	Enabled bool   `json:"enabled"`
	SiteKey string `json:"site_key"`
}

type userInfoResp struct {
	Id          string   `json:"id"`
	Username    string   `json:"username"`
	Email       *string  `json:"email"`
	AuthSource  string   `json:"auth_source"`
	CreatedAt   string   `json:"created_at"`
	LastLoginAt *string  `json:"last_login_at"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type jwtClaims struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
}

func (s Server) registerAuthRoutes(r chiRouter) {
	r.Get("/api/auth/csrf-token", s.getCSRFToken)
	r.Get("/api/auth/turnstile-config", s.getTurnstileConfig)
	r.Post("/api/auth/login", s.login)
	r.Post("/api/auth/logout", s.logout)
	r.Get("/api/auth/me", s.getMe)
	r.Put("/api/auth/password", s.changePassword)
	r.Get("/api/auth/login-history", s.listLoginHistory)
	r.Get("/api/auth/google", s.googleOAuth)
	r.Post("/api/auth/google/callback", s.googleCallback)
}

func (s Server) getCSRFToken(w http.ResponseWriter, r *http.Request) {
	token, err := randomHex(16)
	if err != nil {
		s.logger.Error("generate csrf token failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to generate token"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s Server) getTurnstileConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, turnstileConfigResp{Enabled: s.appCfg.Turnstile.Enabled, SiteKey: s.appCfg.Turnstile.SiteKey})
}

func (s Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if strings.TrimSpace(req.Username) == "" || req.Password == "" || strings.TrimSpace(req.CSRFToken) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Missing required login fields"})
		return
	}
	if s.appCfg.Turnstile.Enabled {
		turnstileToken := strings.TrimSpace(req.TurnstileToken)
		if turnstileToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Missing required login fields"})
			return
		}
		if err := s.turnstileVerifier.Verify(r.Context(), turnstileToken, clientIP(r)); err != nil {
			s.logger.Warn("turnstile verification failed", "error", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid verification"})
			return
		}
	}
	user, err := s.store.UserByUsername(r.Context(), strings.TrimSpace(req.Username))
	if err != nil || user.Status != "enabled" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid username or password"})
		return
	}
	if err := s.store.MarkUserLoggedIn(r.Context(), user.Id); err != nil {
		s.logger.Warn("mark user login time failed", "user_id", user.Id, "error", err)
	}
	if err := s.store.SaveLoginHistory(r.Context(), orbit.LoginHistory{Id: orbit.NewId(), UserId: user.Id, Username: user.Username, IpAddress: stringPtr(clientIP(r)), UserAgent: stringPtr(r.UserAgent()), LoginAt: time.Now().UTC(), Success: true}); err != nil {
		s.logger.Warn("save login history failed", "user_id", user.Id, "error", err)
	}
	token, err := s.signToken(user)
	if err != nil {
		s.logger.Error("sign token failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to sign token"})
		return
	}
	writeJSON(w, http.StatusOK, tokenResp{AccessToken: token, TokenType: "bearer"})
}

func (s Server) logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) getMe(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	roles, err := s.store.UserRoles(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load user roles failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user roles"})
		return
	}
	permissions, err := s.store.UserPermissions(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user permissions"})
		return
	}
	writeJSON(w, http.StatusOK, userInfo(user, roles, permissions))
}

func (s Server) changePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req passwordChangeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if req.OldPassword == "" || len(req.NewPassword) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid password fields"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid username or password"})
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("hash password failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to hash password"})
		return
	}
	user.PasswordHash = string(passwordHash)
	if err := s.store.UpdateUser(r.Context(), user); err != nil {
		s.logger.Error("change password failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to change password"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) listLoginHistory(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "login:read"); !ok {
		return
	}
	page, perPage := pageParams(r)
	history, err := s.store.ListLoginHistory(r.Context(), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list login history failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list login history"})
		return
	}
	resp := mapPage(history, loginHistoryResponse)
	writeJSON(w, http.StatusOK, newPaginatedResp(resp))
}

func (s Server) googleOAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "Google OAuth is not configured"})
}

func (s Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	var req googleCallbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "Google OAuth is not configured"})
}

func (s Server) currentUser(w http.ResponseWriter, r *http.Request) (orbit.User, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Not authenticated"})
		return orbit.User{}, false
	}
	claims, err := s.verifyToken(token)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid token"})
		return orbit.User{}, false
	}
	user, err := s.store.UserById(r.Context(), claims.Sub)
	if err != nil || user.Status != "enabled" {
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.logger.Warn("load current user failed", "user_id", claims.Sub, "error", err)
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid token"})
		return orbit.User{}, false
	}
	return user, true
}

func userInfo(user orbit.User, roles []string, permissions []string) userInfoResp {
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	return userInfoResp{
		Id:          user.Id,
		Username:    user.Username,
		Email:       user.Email,
		AuthSource:  user.AuthSource,
		CreatedAt:   formatTime(user.CreatedAt),
		LastLoginAt: formatOptionalTime(user.LastLoginAt),
		Roles:       roles,
		Permissions: permissions,
	}
}

func (s Server) signToken(user orbit.User) (string, error) {
	claims, err := json.Marshal(jwtClaims{Sub: user.Id, Username: user.Username, Exp: time.Now().Add(24 * time.Hour).Unix()})
	if err != nil {
		return "", err
	}
	return s.signJWT(claims), nil
}

func (s Server) verifyToken(token string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, errors.New("invalid token format")
	}
	signed := parts[0] + "." + parts[1]
	expected := s.jwtSignature(signed)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return jwtClaims{}, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, err
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return jwtClaims{}, err
	}
	if claims.Sub == "" || claims.Exp < time.Now().Unix() {
		return jwtClaims{}, errors.New("token expired")
	}
	return claims, nil
}

func (s Server) signJWT(payload []byte) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	signed := header + "." + body
	return signed + "." + s.jwtSignature(signed)
}

func (s Server) jwtSignature(signed string) string {
	mac := hmac.New(sha256.New, []byte(s.jwtSecret()))
	mac.Write([]byte(signed))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s Server) jwtSecret() string {
	if strings.TrimSpace(s.appCfg.JWT.SecretKey) != "" {
		return s.appCfg.JWT.SecretKey
	}
	return "pomelo-orbit-dev-secret"
}

func clientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func loginHistoryResponse(history orbit.LoginHistory) loginHistoryResp {
	return loginHistoryResp{Id: history.Id, UserId: history.UserId, Username: history.Username, IpAddress: history.IpAddress, UserAgent: history.UserAgent, LoginAt: formatTime(history.LoginAt), Success: history.Success}
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
