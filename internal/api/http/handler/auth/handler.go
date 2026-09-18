package authhandler

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	authsvc "github.com/leoninew/pomelo-orbit/internal/application/auth/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	authv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/auth"
)

type TurnstileVerifier interface {
	Verify(ctx context.Context, token string, remoteIP string) error
}

type Handler struct {
	logger            *slog.Logger
	turnstile         config.TurnstileConfig
	service           authsvc.Service
	authenticator     security.Authenticator
	turnstileVerifier TurnstileVerifier
}

func New(logger *slog.Logger, turnstile config.TurnstileConfig, service authsvc.Service, authenticator security.Authenticator, verifier TurnstileVerifier) Handler {
	return Handler{logger: logger, turnstile: turnstile, service: service, authenticator: authenticator, turnstileVerifier: verifier}
}

func (h Handler) GetCSRFToken(c *gin.Context) {
	token, err := h.service.NewCSRFToken()
	if err != nil {
		h.logger.Error("generate csrf token failed", "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	resp := csrfTokenResponse(token)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) GetTurnstileConfig(c *gin.Context) {
	resp := turnstileConfigResponse(h.turnstile.Enabled, h.turnstile.SiteKey)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) Login(c *gin.Context) {
	var req authv1.LoginReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	input := loginInput(&req, clientIP(c), c.Request.UserAgent())
	if err := authsvc.ValidateLoginInput(input); err != nil {
		transport.WriteError(c, err)
		return
	}
	if h.turnstile.Enabled {
		turnstileToken := strings.TrimSpace(req.TurnstileToken)
		if turnstileToken == "" {
			transport.WriteError(c, authsvc.ErrMissingLoginFields)
			return
		}
		if err := h.turnstileVerifier.Verify(c.Request.Context(), turnstileToken, clientIP(c)); err != nil {
			h.logger.Warn("turnstile verification failed", "error", err)
			transport.WriteStatusError(c, http.StatusBadRequest, "Invalid verification")
			return
		}
	}
	token, err := h.service.Login(c.Request.Context(), input)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := tokenResponse(token)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) Logout(c *gin.Context) {
	var req authv1.LogoutReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) GetMe(c *gin.Context) {
	user, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	roles, err := h.service.UserRoles(c.Request.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user roles failed", "user_id", user.Id, "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	permissions, err := h.service.UserPermissions(c.Request.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	resp := userInfoResponse(user, roles, permissions)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ChangePassword(c *gin.Context) {
	user, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req authv1.PasswordChangeReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), changePasswordInput(user, &req)); err != nil {
		if apperror.IsKind(err, apperror.KindValidation) || apperror.IsKind(err, apperror.KindUnauthorized) {
			transport.WriteError(c, err)
			return
		}
		h.logger.Error("change password failed", "user_id", user.Id, "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ListLoginHistory(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "login:read"); !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	history, err := h.service.ListLoginHistory(c.Request.Context(), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list login history failed", "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	items := make([]authv1.LoginHistoryResp, 0, len(history.Items))
	for _, item := range history.Items {
		items = append(items, loginHistoryResponse(item))
	}
	transport.WriteProtoJSON(c, http.StatusOK, &authv1.LoginHistoryPaginatedResp{Items: transport.Ptrs(items), Total: int32(history.Total), Page: int32(history.Page), PerPage: int32(history.PerPage), Pages: int32(transport.PageCount(history.Total, history.PerPage))})
}

func (h Handler) ListMCPAccessTokens(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListMCPAccessTokens(c.Request.Context(), current.Id)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	response := make([]authv1.MCPAccessTokenResp, 0, len(items))
	for _, item := range items {
		response = append(response, mcpAccessTokenResponse(item))
	}
	transport.WriteProtoJSON(c, http.StatusOK, &authv1.MCPAccessTokenListResp{Items: transport.Ptrs(response)})
}

func (h Handler) CreateMCPAccessToken(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req authv1.MCPAccessTokenCreateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	created, err := h.service.CreateMCPAccessToken(c.Request.Context(), current.Id, authdto.MCPAccessTokenCreateInput{Name: req.Name, ExpiresInDays: req.ExpiresInDays})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	accessToken := mcpAccessTokenResponse(created.AccessToken)
	resp := authv1.MCPAccessTokenCreatedResp{AccessToken: &accessToken, Token: created.Token}
	transport.WriteProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) RevokeMCPAccessToken(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.RevokeMCPAccessToken(c.Request.Context(), current.Id, c.Param("token_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) GoogleOAuth(c *gin.Context) {
	transport.WriteStatusError(c, http.StatusServiceUnavailable, "Google OAuth is not configured")
}

func (h Handler) GoogleCallback(c *gin.Context) {
	var req authv1.GoogleCallbackReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	transport.WriteStatusError(c, http.StatusServiceUnavailable, "Google OAuth is not configured")
}

func clientIP(c *gin.Context) string {
	forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	realIP := strings.TrimSpace(c.GetHeader("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
