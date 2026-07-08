package rolehandler

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
}

type Store interface {
	ListRoles(ctx context.Context, page int, perPage int, search string) (repository.Page[model.Role], error)
	ListPermissions(ctx context.Context) ([]model.Permission, error)
	RoleById(ctx context.Context, id string) (model.Role, error)
	RolePermissionCodesByRoleIds(ctx context.Context, roleIds []string) (map[string][]string, error)
}

type Handler struct {
	logger        *slog.Logger
	service       rolesvc.Service
	authenticator authz.Authenticator
	store         Store
}

func New(logger *slog.Logger, service rolesvc.Service, authenticator authz.Authenticator, store Store) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator, store: store}
}

func (h Handler) Register(r router) {
	r.GET("/api/role", h.listRoles)
	r.POST("/api/role", h.createRole)
	r.GET("/api/role/permission", h.listPermissions)
	r.GET("/api/role/:role_id", h.getRole)
	r.PUT("/api/role/:role_id", h.updateRole)
	r.DELETE("/api/role/:role_id", h.deleteRole)
}

func (h Handler) listRoles(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	roles, err := h.store.ListRoles(c.Request.Context(), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list roles failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to list roles"})
		return
	}
	roleIds := make([]string, 0, len(roles.Items))
	for _, role := range roles.Items {
		roleIds = append(roleIds, role.Id)
	}
	permissionsByRoleId, err := h.store.RolePermissionCodesByRoleIds(c.Request.Context(), roleIds)
	if err != nil {
		h.logger.Error("load role permissions failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to load role permissions"})
		return
	}
	resp := mapPage(roles, func(role model.Role) pomeloorbit.RoleResp {
		return roleResponse(role, permissionsByRoleId[role.Id])
	})
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RolePaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) listPermissions(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	permissions, err := h.store.ListPermissions(c.Request.Context())
	if err != nil {
		h.logger.Error("list permissions failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to list permissions"})
		return
	}
	items := make([]pomeloorbit.PermissionResp, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, permissionResponse(permission))
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.PermissionListResp{Items: transportresponse.Ptrs(items)}})
}

func (h Handler) getRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:read"); !ok {
		return
	}
	role, ok := h.loadRoleFromPath(c)
	if !ok {
		return
	}
	resp, ok := h.roleDetailResp(c, role)
	if !ok {
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) createRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	var req pomeloorbit.RoleCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	role, err := h.service.Create(c.Request.Context(), rolesvc.SaveInput{Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes})
	if err != nil {
		h.writeServiceError(c, err, "Failed to create role")
		return
	}
	resp := roleResponse(role, req.PermissionCodes)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	role, ok := h.loadRoleFromPath(c)
	if !ok {
		return
	}
	var req pomeloorbit.RoleUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	updated, err := h.service.Update(c.Request.Context(), rolesvc.SaveInput{Role: role, Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes})
	if err != nil {
		h.writeServiceError(c, err, "Failed to update role")
		return
	}
	resp, ok := h.roleDetailResp(c, updated)
	if !ok {
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteRole(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "role:write"); !ok {
		return
	}
	role, ok := h.loadRoleFromPath(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), role.Id); err != nil {
		h.logger.Error("delete role failed", "role_id", role.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to delete role"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) writeServiceError(c *gin.Context, err error, internalDetail string) {
	if apperror.IsKind(err, apperror.KindValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	if apperror.IsKind(err, apperror.KindConflict) {
		c.JSON(http.StatusConflict, gin.H{"detail": err.Error()})
		return
	}
	if apperror.IsKind(err, apperror.KindNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}
	h.logger.Error("role service failed", "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"detail": internalDetail})
}

func (h Handler) loadRoleFromPath(c *gin.Context) (model.Role, bool) {
	roleId := strings.TrimSpace(c.Param("role_id"))
	role, err := h.store.RoleById(c.Request.Context(), roleId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Role " + roleId + " not found"})
			return model.Role{}, false
		}
		h.logger.Error("load role failed", "role_id", roleId, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to load role"})
		return model.Role{}, false
	}
	return role, true
}

func (h Handler) roleDetailResp(c *gin.Context, role model.Role) (pomeloorbit.RoleResp, bool) {
	permissionsByRoleId, err := h.store.RolePermissionCodesByRoleIds(c.Request.Context(), []string{role.Id})
	if err != nil {
		h.logger.Error("load role permissions failed", "role_id", role.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to load role permissions"})
		return pomeloorbit.RoleResp{}, false
	}
	return roleResponse(role, permissionsByRoleId[role.Id]), true
}

func roleResponse(role model.Role, permissionCodes []string) pomeloorbit.RoleResp {
	if permissionCodes == nil {
		permissionCodes = []string{}
	}
	return pomeloorbit.RoleResp{Id: role.Id, Code: role.Code, Name: role.Name, Description: role.Description, CreatedAt: transportresponse.FormatTime(role.CreatedAt), UpdatedAt: transportresponse.FormatTime(role.UpdatedAt), PermissionCodes: permissionCodes}
}

func permissionResponse(permission model.Permission) pomeloorbit.PermissionResp {
	return pomeloorbit.PermissionResp{Id: permission.Id, Code: permission.Code, Name: permission.Name, Description: permission.Description}
}

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}
