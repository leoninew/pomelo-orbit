package rolehandler

import (
	"log/slog"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger        *slog.Logger
	service       rolesvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service rolesvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) ListRoles(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	roles, err := h.service.List(c.Request.Context(), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list roles failed", "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to list roles")
		return
	}
	items := make([]pomeloorbit.RoleResp, 0, len(roles.Items))
	for _, role := range roles.Items {
		items = append(items, roleDetailResponse(role))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.RolePaginatedResp{Items: transportresponse.Ptrs(items), Total: int32(roles.Total), Page: int32(roles.Page), PerPage: int32(roles.PerPage), Pages: int32(transportresponse.PageCount(roles.Total, roles.PerPage))})
}

func (h Handler) ListPermissions(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		h.logger.Error("list permissions failed", "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to list permissions")
		return
	}
	items := make([]pomeloorbit.PermissionResp, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, permissionResponse(permission))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.PermissionListResp{Items: transportresponse.Ptrs(items)})
}

func (h Handler) GetRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	detail, err := h.service.Detail(c.Request.Context(), roleID(c))
	if err != nil {
		h.writeServiceError(c, err, "Failed to load role")
		return
	}
	resp := roleDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) CreateRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	var req pomeloorbit.RoleCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	role, err := h.service.Create(c.Request.Context(), roleCreateInput(&req))
	if err != nil {
		h.writeServiceError(c, err, "Failed to create role")
		return
	}
	resp := roleResponse(role, req.PermissionCodes)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) UpdateRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	var req pomeloorbit.RoleUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.UpdateByID(c.Request.Context(), roleID(c), roleUpdateInput(&req))
	if err != nil {
		h.writeServiceError(c, err, "Failed to update role")
		return
	}
	resp := roleDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), roleID(c)); err != nil {
		h.writeServiceError(c, err, "Failed to delete role")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) writeServiceError(c *gin.Context, err error, internalDetail string) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindConflict) {
		transportresponse.Error(c, http.StatusConflict, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindNotFound) {
		transportresponse.Error(c, http.StatusNotFound, err.Error())
		return
	}
	h.logger.Error("role service failed", "error", err)
	transportresponse.Error(c, http.StatusInternalServerError, internalDetail)
}

func roleID(c *gin.Context) string {
	return strings.TrimSpace(c.Param("role_id"))
}
