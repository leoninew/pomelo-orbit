package authhandler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/repository/model"
	authsvc "backend/internal/service/auth"
	"backend/internal/transport/http/handler/authz"
	transportresponse "backend/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
}

type Store interface {
	UserRoles(ctx context.Context, userId string) ([]string, error)
	UserPermissions(ctx context.Context, userId string) ([]string, error)
}

type TurnstileVerifier interface {
	Verify(ctx context.Context, token string, remoteIP string) error
}

type Handler struct {
	logger            *slog.Logger
	turnstile         config.TurnstileConfig
	service           authsvc.Service
	authenticator     authz.Authenticator
	turnstileVerifier TurnstileVerifier
	store             Store
}

type LoginHistoryResp struct {
	Id        string  `json:"id"`
	UserId    string  `json:"user_id"`
	Username  string  `json:"username"`
	IpAddress *string `json:"ip_address"`
	UserAgent *string `json:"user_agent"`
	LoginAt   string  `json:"login_at"`
	Success   bool    `json:"success"`
}

type TokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type UserInfoResp struct {
	Id          string   `json:"id"`
	Username    string   `json:"username"`
	Email       *string  `json:"email"`
	AuthSource  string   `json:"auth_source"`
	CreatedAt   string   `json:"created_at"`
	LastLoginAt *string  `json:"last_login_at"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

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

type turnstileConfigResp struct {
	Enabled bool   `json:"enabled"`
	SiteKey string `json:"site_key"`
}

func New(logger *slog.Logger, turnstile config.TurnstileConfig, service authsvc.Service, authenticator authz.Authenticator, verifier TurnstileVerifier, store Store) Handler {
	return Handler{logger: logger, turnstile: turnstile, service: service, authenticator: authenticator, turnstileVerifier: verifier, store: store}
}

func (h Handler) Register(r router) {
	r.Get("/api/auth/csrf-token", h.getCSRFToken)
	r.Get("/api/auth/turnstile-config", h.getTurnstileConfig)
	r.Post("/api/auth/login", h.login)
	r.Post("/api/auth/logout", h.logout)
	r.Get("/api/auth/me", h.getMe)
	r.Put("/api/auth/password", h.changePassword)
	r.Get("/api/auth/login-history", h.listLoginHistory)
	r.Get("/api/auth/google", h.googleOAuth)
	r.Post("/api/auth/google/callback", h.googleCallback)
}

func (h Handler) getCSRFToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.service.NewCSRFToken()
	if err != nil {
		h.logger.Error("generate csrf token failed", "error", err)
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to generate token"})
		return
	}
	transportresponse.JSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h Handler) getTurnstileConfig(w http.ResponseWriter, r *http.Request) {
	transportresponse.JSON(w, http.StatusOK, turnstileConfigResp{Enabled: h.turnstile.Enabled, SiteKey: h.turnstile.SiteKey})
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	loginInput := authsvc.LoginInput{Username: req.Username, Password: req.Password, CSRFToken: req.CSRFToken, IP: clientIP(r), UserAgent: r.UserAgent()}
	if err := authsvc.ValidateLoginInput(loginInput); err != nil {
		writeServiceError(w, err)
		return
	}
	if h.turnstile.Enabled {
		turnstileToken := strings.TrimSpace(req.TurnstileToken)
		if turnstileToken == "" {
			writeServiceError(w, authsvc.ErrMissingLoginFields)
			return
		}
		if err := h.turnstileVerifier.Verify(r.Context(), turnstileToken, clientIP(r)); err != nil {
			h.logger.Warn("turnstile verification failed", "error", err)
			transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid verification"})
			return
		}
	}
	token, err := h.service.Login(r.Context(), loginInput)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, TokenResp{AccessToken: token, TokenType: "bearer"})
}

func (h Handler) logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) getMe(w http.ResponseWriter, r *http.Request) {
	user, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	roles, err := h.store.UserRoles(r.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user roles failed", "user_id", user.Id, "error", err)
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user roles"})
		return
	}
	permissions, err := h.store.UserPermissions(r.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user permissions"})
		return
	}
	transportresponse.JSON(w, http.StatusOK, userInfo(user, roles, permissions))
}

func (h Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req passwordChangeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if err := h.service.ChangePassword(r.Context(), authsvc.ChangePasswordInput{User: user, OldPassword: req.OldPassword, NewPassword: req.NewPassword}); err != nil {
		if apperror.IsKind(err, apperror.KindValidation) || apperror.IsKind(err, apperror.KindUnauthorized) {
			writeServiceError(w, err)
			return
		}
		h.logger.Error("change password failed", "user_id", user.Id, "error", err)
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to change password"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) listLoginHistory(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "login:read"); !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	history, err := h.service.ListLoginHistory(r.Context(), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list login history failed", "error", err)
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list login history"})
		return
	}
	resp := mapPage(history, loginHistoryResponse)
	transportresponse.JSON(w, http.StatusOK, transportresponse.NewPaginatedResp(resp))
}

func (h Handler) googleOAuth(w http.ResponseWriter, r *http.Request) {
	transportresponse.JSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "Google OAuth is not configured"})
}

func (h Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	var req googleCallbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	transportresponse.JSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "Google OAuth is not configured"})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
		return
	}
	if apperror.IsKind(err, apperror.KindUnauthorized) {
		transportresponse.JSON(w, http.StatusUnauthorized, map[string]string{"detail": err.Error()})
		return
	}
	transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to sign token"})
}

func userInfo(user model.User, roles []string, permissions []string) UserInfoResp {
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	return UserInfoResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roles, Permissions: permissions}
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

func loginHistoryResponse(history model.LoginHistory) LoginHistoryResp {
	return LoginHistoryResp{Id: history.Id, UserId: history.UserId, Username: history.Username, IpAddress: history.IpAddress, UserAgent: history.UserAgent, LoginAt: transportresponse.FormatTime(history.LoginAt), Success: history.Success}
}

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}
