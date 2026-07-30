package authhandler

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	authv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/auth"
	"github.com/gin-gonic/gin"
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
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	resp := csrfTokenResponse(token)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) GetTurnstileConfig(c *gin.Context) {
	resp := turnstileConfigResponse(h.turnstile.Enabled, h.turnstile.SiteKey)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) Login(c *gin.Context) {
	var req authv1.LoginReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	input := loginInput(&req, clientIP(c), c.Request.UserAgent())
	if err := authsvc.ValidateLoginInput(input); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	if h.turnstile.Enabled {
		turnstileToken := strings.TrimSpace(req.TurnstileToken)
		if turnstileToken == "" {
			transportresponse.WriteError(c, authsvc.ErrMissingLoginFields)
			return
		}
		if err := h.turnstileVerifier.Verify(c.Request.Context(), turnstileToken, clientIP(c)); err != nil {
			h.logger.Warn("turnstile verification failed", "error", err)
			transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid verification")
			return
		}
	}
	token, err := h.service.Login(c.Request.Context(), input)
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := tokenResponse(token)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) Logout(c *gin.Context) {
	var req authv1.LogoutReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
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
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	permissions, err := h.service.UserPermissions(c.Request.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	resp := userInfoResponse(user, roles, permissions)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ChangePassword(c *gin.Context) {
	user, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req authv1.PasswordChangeReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), changePasswordInput(user, &req)); err != nil {
		if apperror.IsKind(err, apperror.KindValidation) || apperror.IsKind(err, apperror.KindUnauthorized) {
			transportresponse.WriteError(c, err)
			return
		}
		h.logger.Error("change password failed", "user_id", user.Id, "error", err)
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ListLoginHistory(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "login:read"); !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	history, err := h.service.ListLoginHistory(c.Request.Context(), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list login history failed", "error", err)
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	items := make([]authv1.LoginHistoryResp, 0, len(history.Items))
	for _, item := range history.Items {
		items = append(items, loginHistoryResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &authv1.LoginHistoryPaginatedResp{Items: transportresponse.Ptrs(items), Total: int32(history.Total), Page: int32(history.Page), PerPage: int32(history.PerPage), Pages: int32(transportresponse.PageCount(history.Total, history.PerPage))})
}

func (h Handler) GoogleOAuth(c *gin.Context) {
	transportresponse.WriteStatusError(c, http.StatusServiceUnavailable, "Google OAuth is not configured")
}

func (h Handler) GoogleCallback(c *gin.Context) {
	var req authv1.GoogleCallbackReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	transportresponse.WriteStatusError(c, http.StatusServiceUnavailable, "Google OAuth is not configured")
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
