package authhandler

import (
	"context"
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
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

func New(logger *slog.Logger, turnstile config.TurnstileConfig, service authsvc.Service, authenticator authz.Authenticator, verifier TurnstileVerifier, store Store) Handler {
	return Handler{logger: logger, turnstile: turnstile, service: service, authenticator: authenticator, turnstileVerifier: verifier, store: store}
}

func (h Handler) Register(r router) {
	r.GET("/api/auth/csrf-token", h.getCSRFToken)
	r.GET("/api/auth/turnstile-config", h.getTurnstileConfig)
	r.POST("/api/auth/login", h.login)
	r.POST("/api/auth/logout", h.logout)
	r.GET("/api/auth/me", h.getMe)
	r.PUT("/api/auth/password", h.changePassword)
	r.GET("/api/auth/login-history", h.listLoginHistory)
	r.GET("/api/auth/google", h.googleOAuth)
	r.POST("/api/auth/google/callback", h.googleCallback)
}

func (h Handler) getCSRFToken(c *gin.Context) {
	token, err := h.service.NewCSRFToken()
	if err != nil {
		h.logger.Error("generate csrf token failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to generate token"})
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.CSRFTokenResp{Token: token}})
}

func (h Handler) getTurnstileConfig(c *gin.Context) {
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.TurnstileConfigResp{Enabled: h.turnstile.Enabled, SiteKey: h.turnstile.SiteKey}})
}

func (h Handler) login(c *gin.Context) {
	var req pomeloorbit.LoginReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	loginInput := authsvc.LoginInput{Username: req.Username, Password: req.Password, CSRFToken: req.CsrfToken, IP: clientIP(c), UserAgent: c.Request.UserAgent()}
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
			c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid verification"})
			return
		}
	}
	token, err := h.service.Login(c.Request.Context(), loginInput)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.TokenResp{AccessToken: token, TokenType: "bearer"}})
}

func (h Handler) logout(c *gin.Context) {
	var req pomeloorbit.LogoutReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) getMe(c *gin.Context) {
	user, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	roles, err := h.store.UserRoles(c.Request.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user roles failed", "user_id", user.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to load user roles"})
		return
	}
	permissions, err := h.store.UserPermissions(c.Request.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to load user permissions"})
		return
	}
	resp := userInfo(user, roles, permissions)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) changePassword(c *gin.Context) {
	user, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PasswordChangeReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), authsvc.ChangePasswordInput{User: user, OldPassword: req.OldPassword, NewPassword: req.NewPassword}); err != nil {
		if apperror.IsKind(err, apperror.KindValidation) || apperror.IsKind(err, apperror.KindUnauthorized) {
			h.writeServiceError(c, err)
			return
		}
		h.logger.Error("change password failed", "user_id", user.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to change password"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) listLoginHistory(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "login:read"); !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	history, err := h.service.ListLoginHistory(c.Request.Context(), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list login history failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to list login history"})
		return
	}
	resp := mapPage(history, loginHistoryResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.LoginHistoryPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) googleOAuth(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"detail": "Google OAuth is not configured"})
}

func (h Handler) googleCallback(c *gin.Context) {
	var req pomeloorbit.GoogleCallbackReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"detail": "Google OAuth is not configured"})
}

func (h Handler) writeServiceError(c *gin.Context, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	if apperror.IsKind(err, apperror.KindUnauthorized) {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to sign token"})
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

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}
