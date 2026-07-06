package rolehandler

import (
	"context"
	"database/sql"
	"errors"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gitee.com/leoninew/PomeloOrbit-go/internal/apperror"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/role"
	"gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
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
	r.Get("/api/role", h.listRoles)
	r.Post("/api/role", h.createRole)
	r.Get("/api/role/permission", h.listPermissions)
	r.Get("/api/role/{role_id}", h.getRole)
	r.Put("/api/role/{role_id}", h.updateRole)
	r.Delete("/api/role/{role_id}", h.deleteRole)
}

func (h Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "role:read"); !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	roles, err := h.store.ListRoles(r.Context(), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list roles failed", "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to list roles")
		return
	}
	roleIds := make([]string, 0, len(roles.Items))
	for _, role := range roles.Items {
		roleIds = append(roleIds, role.Id)
	}
	permissionsByRoleId, err := h.store.RolePermissionCodesByRoleIds(r.Context(), roleIds)
	if err != nil {
		h.logger.Error("load role permissions failed", "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load role permissions")
		return
	}
	resp := mapPage(roles, func(role model.Role) pomeloorbit.RoleResp {
		return roleResponse(role, permissionsByRoleId[role.Id])
	})
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.RolePaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "role:read"); !ok {
		return
	}
	permissions, err := h.store.ListPermissions(r.Context())
	if err != nil {
		h.logger.Error("list permissions failed", "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to list permissions")
		return
	}
	items := make([]pomeloorbit.PermissionResp, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, permissionResponse(permission))
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.PermissionListResp{Items: transportresponse.Ptrs(items)})
}

func (h Handler) getRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "role:read"); !ok {
		return
	}
	role, ok := h.loadRoleFromPath(w, r)
	if !ok {
		return
	}
	resp, ok := h.roleDetailResp(w, r, role)
	if !ok {
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) createRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "role:write"); !ok {
		return
	}
	var req pomeloorbit.RoleCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	role, err := h.service.Create(r.Context(), rolesvc.SaveInput{Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes})
	if err != nil {
		h.writeServiceError(w, err, "Failed to create role")
		return
	}
	resp := roleResponse(role, req.PermissionCodes)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "role:write"); !ok {
		return
	}
	role, ok := h.loadRoleFromPath(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.RoleUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	updated, err := h.service.Update(r.Context(), rolesvc.SaveInput{Role: role, Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes})
	if err != nil {
		h.writeServiceError(w, err, "Failed to update role")
		return
	}
	resp, ok := h.roleDetailResp(w, r, updated)
	if !ok {
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "role:write"); !ok {
		return
	}
	role, ok := h.loadRoleFromPath(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), role.Id); err != nil {
		h.logger.Error("delete role failed", "role_id", role.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to delete role")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) writeServiceError(w http.ResponseWriter, err error, internalDetail string) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindConflict) {
		transportresponse.Error(h.logger, w, http.StatusConflict, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindNotFound) {
		transportresponse.Error(h.logger, w, http.StatusNotFound, err.Error())
		return
	}
	h.logger.Error("role service failed", "error", err)
	transportresponse.Error(h.logger, w, http.StatusInternalServerError, internalDetail)
}

func (h Handler) loadRoleFromPath(w http.ResponseWriter, r *http.Request) (model.Role, bool) {
	roleId := strings.TrimSpace(chi.URLParam(r, "role_id"))
	role, err := h.store.RoleById(r.Context(), roleId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			transportresponse.Error(h.logger, w, http.StatusNotFound, "Role "+roleId+" not found")
			return model.Role{}, false
		}
		h.logger.Error("load role failed", "role_id", roleId, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load role")
		return model.Role{}, false
	}
	return role, true
}

func (h Handler) roleDetailResp(w http.ResponseWriter, r *http.Request, role model.Role) (pomeloorbit.RoleResp, bool) {
	permissionsByRoleId, err := h.store.RolePermissionCodesByRoleIds(r.Context(), []string{role.Id})
	if err != nil {
		h.logger.Error("load role permissions failed", "role_id", role.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load role permissions")
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
