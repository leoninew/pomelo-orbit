package authhandler

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	authdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/dto"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"github.com/gin-gonic/gin"
)

type TurnstileVerifier interface {
	Verify(ctx context.Context, token string, remoteIP string) error
}

type Handler struct {
	logger            *slog.Logger
	turnstile         config.TurnstileConfig
	service           authsvc.Service
	authenticator     authz.Authenticator
	turnstileVerifier TurnstileVerifier
}

func New(logger *slog.Logger, turnstile config.TurnstileConfig, service authsvc.Service, authenticator authz.Authenticator, verifier TurnstileVerifier) Handler {
	return Handler{logger: logger, turnstile: turnstile, service: service, authenticator: authenticator, turnstileVerifier: verifier}
}

func (h Handler) GetCSRFToken(c *gin.Context) {
	token, err := h.service.NewCSRFToken()
	if err != nil {
		h.logger.Error("generate csrf token failed", "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.CSRFTokenResp{Token: token})
}

func (h Handler) GetTurnstileConfig(c *gin.Context) {
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.TurnstileConfigResp{Enabled: h.turnstile.Enabled, SiteKey: h.turnstile.SiteKey})
}

func (h Handler) Login(c *gin.Context) {
	var req pomeloorbit.LoginReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	loginInput := authdto.LoginInput{Username: req.Username, Password: req.Password, CSRFToken: req.CsrfToken, IP: clientIP(c), UserAgent: c.Request.UserAgent()}
	if err := authsvc.ValidateLoginInput(loginInput); err != nil {
		h.writeServiceError(c, err)
		return
	}
	if h.turnstile.Enabled {
		turnstileToken := strings.TrimSpace(req.TurnstileToken)
		if turnstileToken == "" {
			h.writeServiceError(c, authsvc.ErrMissingLoginFields)
			return
		}
		if err := h.turnstileVerifier.Verify(c.Request.Context(), turnstileToken, clientIP(c)); err != nil {
			h.logger.Warn("turnstile verification failed", "error", err)
			transportresponse.Error(c, http.StatusBadRequest, "Invalid verification")
			return
		}
	}
	token, err := h.service.Login(c.Request.Context(), loginInput)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.TokenResp{AccessToken: token, TokenType: "bearer"})
}

func (h Handler) Logout(c *gin.Context) {
	var req pomeloorbit.LogoutReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
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
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to load user roles")
		return
	}
	permissions, err := h.service.UserPermissions(c.Request.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to load user permissions")
		return
	}
	resp := userInfo(user, roles, permissions)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ChangePassword(c *gin.Context) {
	user, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PasswordChangeReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), authdto.ChangePasswordInput{User: user, OldPassword: req.OldPassword, NewPassword: req.NewPassword}); err != nil {
		if apperror.IsKind(err, apperror.KindValidation) || apperror.IsKind(err, apperror.KindUnauthorized) {
			h.writeServiceError(c, err)
			return
		}
		h.logger.Error("change password failed", "user_id", user.Id, "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to change password")
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
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to list login history")
		return
	}
	items := make([]pomeloorbit.LoginHistoryResp, 0, len(history.Items))
	for _, item := range history.Items {
		items = append(items, loginHistoryResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.LoginHistoryPaginatedResp{Items: transportresponse.Ptrs(items), Total: int32(history.Total), Page: int32(history.Page), PerPage: int32(history.PerPage), Pages: int32(transportresponse.PageCount(history.Total, history.PerPage))})
}

func (h Handler) GoogleOAuth(c *gin.Context) {
	transportresponse.Error(c, http.StatusServiceUnavailable, "Google OAuth is not configured")
}

func (h Handler) GoogleCallback(c *gin.Context) {
	var req pomeloorbit.GoogleCallbackReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	transportresponse.Error(c, http.StatusServiceUnavailable, "Google OAuth is not configured")
}

func (h Handler) writeServiceError(c *gin.Context, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindUnauthorized) {
		transportresponse.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	transportresponse.Error(c, http.StatusInternalServerError, "Failed to sign token")
}

func userInfo(user model.User, roles []string, permissions []string) pomeloorbit.UserInfoResp {
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	return pomeloorbit.UserInfoResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roles, Permissions: permissions}
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

func loginHistoryResponse(history model.LoginHistory) pomeloorbit.LoginHistoryResp {
	return pomeloorbit.LoginHistoryResp{Id: history.Id, UserId: history.UserId, Username: history.Username, IpAddress: history.IpAddress, UserAgent: history.UserAgent, LoginAt: transportresponse.FormatTime(history.LoginAt), Success: history.Success}
}
