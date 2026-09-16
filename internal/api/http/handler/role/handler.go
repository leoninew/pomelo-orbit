package rolehandler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	rolesvc "github.com/leoninew/pomelo-orbit/internal/application/role/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	rolev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/role"
)

type Handler struct {
	logger        *slog.Logger
	service       rolesvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service rolesvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) ListRoles(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	roles, err := h.service.List(c.Request.Context(), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list roles failed", "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	items := make([]rolev1.RoleResp, 0, len(roles.Items))
	for _, role := range roles.Items {
		items = append(items, roleDetailResponse(role))
	}
	transport.WriteProtoJSON(c, http.StatusOK, &rolev1.RolePaginatedResp{Items: transport.Ptrs(items), Total: int32(roles.Total), Page: int32(roles.Page), PerPage: int32(roles.PerPage), Pages: int32(transport.PageCount(roles.Total, roles.PerPage))})
}

func (h Handler) ListPermissions(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		h.logger.Error("list permissions failed", "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	items := make([]rolev1.PermissionResp, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, permissionResponse(permission))
	}
	transport.WriteProtoJSON(c, http.StatusOK, &rolev1.PermissionListResp{Items: transport.Ptrs(items)})
}

func (h Handler) GetRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	detail, err := h.service.Detail(c.Request.Context(), roleId(c))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := roleDetailResponse(detail)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) CreateRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	var req rolev1.RoleCreateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	role, err := h.service.Create(c.Request.Context(), roleCreateInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := roleResponse(role, req.PermissionCodes)
	transport.WriteProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) UpdateRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	var req rolev1.RoleUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.UpdateById(c.Request.Context(), roleId(c), roleUpdateInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := roleDetailResponse(detail)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), roleId(c)); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func roleId(c *gin.Context) string {
	return strings.TrimSpace(c.Param("role_id"))
}
