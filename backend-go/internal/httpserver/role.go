package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"backend/internal/orbit"
)

var roleCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type permissionResp struct {
	Id          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type roleResp struct {
	Id              string   `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	PermissionCodes []string `json:"permission_codes"`
}

type roleCreateReq struct {
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
}

type roleUpdateReq struct {
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
}

func (s Server) registerRoleRoutes(r chiRouter) {
	r.Get("/api/role", s.listRoles)
	r.Post("/api/role", s.createRole)
	r.Get("/api/role/permission", s.listPermissions)
	r.Get("/api/role/{role_id}", s.getRole)
	r.Put("/api/role/{role_id}", s.updateRole)
	r.Delete("/api/role/{role_id}", s.deleteRole)
}

func (s Server) listRoles(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "role:read"); !ok {
		return
	}
	page, perPage := pageParams(r)
	roles, err := s.store.ListRoles(r.Context(), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list roles failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list roles"})
		return
	}
	roleIds := make([]string, 0, len(roles.Items))
	for _, role := range roles.Items {
		roleIds = append(roleIds, role.Id)
	}
	permissionsByRoleId, err := s.store.RolePermissionCodesByRoleIds(r.Context(), roleIds)
	if err != nil {
		s.logger.Error("load role permissions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load role permissions"})
		return
	}
	resp := mapPage(roles, func(role orbit.Role) roleResp {
		return roleResponse(role, permissionsByRoleId[role.Id])
	})
	writeJSON(w, http.StatusOK, newPaginatedResp(resp))
}

func (s Server) listPermissions(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "role:read"); !ok {
		return
	}
	permissions, err := s.store.ListPermissions(r.Context())
	if err != nil {
		s.logger.Error("list permissions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list permissions"})
		return
	}
	items := make([]permissionResp, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, permissionResponse(permission))
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) getRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "role:read"); !ok {
		return
	}
	role, ok := s.loadRoleFromPath(w, r)
	if !ok {
		return
	}
	resp, ok := s.roleDetailResp(w, r, role)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) createRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "role:write"); !ok {
		return
	}
	var req roleCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeAndValidateRoleReq(w, &req.Code, &req.Name, req.Description, req.PermissionCodes) {
		return
	}
	if !s.ensureRoleCodeAvailable(w, r, req.Code, "") || !s.ensureRoleNameAvailable(w, r, req.Name, "") || !s.ensurePermissionCodesExist(w, r, req.PermissionCodes) {
		return
	}
	now := time.Now().UTC()
	role := orbit.Role{Id: orbit.NewId(), Code: req.Code, Name: req.Name, Description: req.Description, CreatedAt: now, UpdatedAt: now}
	if err := s.store.CreateRole(r.Context(), role, req.PermissionCodes); err != nil {
		s.logger.Error("create role failed", "role_code", role.Code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create role"})
		return
	}
	writeJSON(w, http.StatusCreated, roleResponse(role, req.PermissionCodes))
}

func (s Server) updateRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "role:write"); !ok {
		return
	}
	role, ok := s.loadRoleFromPath(w, r)
	if !ok {
		return
	}
	var req roleUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeAndValidateRoleReq(w, &req.Code, &req.Name, req.Description, req.PermissionCodes) {
		return
	}
	if !s.ensureRoleCodeAvailable(w, r, req.Code, role.Id) || !s.ensureRoleNameAvailable(w, r, req.Name, role.Id) || !s.ensurePermissionCodesExist(w, r, req.PermissionCodes) {
		return
	}
	role.Code = req.Code
	role.Name = req.Name
	role.Description = req.Description
	if err := s.store.UpdateRole(r.Context(), role, req.PermissionCodes); err != nil {
		s.logger.Error("update role failed", "role_id", role.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update role"})
		return
	}
	updated, err := s.store.RoleById(r.Context(), role.Id)
	if err != nil {
		s.logger.Error("load updated role failed", "role_id", role.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load role"})
		return
	}
	resp, ok := s.roleDetailResp(w, r, updated)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) deleteRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "role:write"); !ok {
		return
	}
	role, ok := s.loadRoleFromPath(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteRole(r.Context(), role.Id); err != nil {
		s.logger.Error("delete role failed", "role_id", role.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete role"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) loadRoleFromPath(w http.ResponseWriter, r *http.Request) (orbit.Role, bool) {
	roleId := urlParam(r, "role_id")
	role, err := s.store.RoleById(r.Context(), roleId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Role " + roleId + " not found"})
			return orbit.Role{}, false
		}
		s.logger.Error("load role failed", "role_id", roleId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load role"})
		return orbit.Role{}, false
	}
	return role, true
}

func (s Server) roleDetailResp(w http.ResponseWriter, r *http.Request, role orbit.Role) (roleResp, bool) {
	permissionsByRoleId, err := s.store.RolePermissionCodesByRoleIds(r.Context(), []string{role.Id})
	if err != nil {
		s.logger.Error("load role permissions failed", "role_id", role.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load role permissions"})
		return roleResp{}, false
	}
	return roleResponse(role, permissionsByRoleId[role.Id]), true
}

func (s Server) ensureRoleCodeAvailable(w http.ResponseWriter, r *http.Request, code string, currentRoleId string) bool {
	existing, err := s.store.RoleByCode(r.Context(), code)
	if err == nil {
		if existing.Id != currentRoleId {
			writeJSON(w, http.StatusConflict, map[string]string{"detail": "Role code " + code + " already exists"})
			return false
		}
		return true
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check role code failed", "role_code", code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check role code"})
		return false
	}
	return true
}

func (s Server) ensureRoleNameAvailable(w http.ResponseWriter, r *http.Request, name string, currentRoleId string) bool {
	existing, err := s.store.RoleByName(r.Context(), name)
	if err == nil {
		if existing.Id != currentRoleId {
			writeJSON(w, http.StatusConflict, map[string]string{"detail": "Role name " + name + " already exists"})
			return false
		}
		return true
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check role name failed", "role_name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check role name"})
		return false
	}
	return true
}

func (s Server) ensurePermissionCodesExist(w http.ResponseWriter, r *http.Request, permissionCodes []string) bool {
	found, err := s.store.PermissionCodesExist(r.Context(), permissionCodes)
	if err != nil {
		s.logger.Error("check permissions failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check permissions"})
		return false
	}
	for _, code := range permissionCodes {
		if _, ok := found[code]; !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Permission " + code + " not found"})
			return false
		}
	}
	return true
}

func normalizeAndValidateRoleReq(w http.ResponseWriter, code *string, name *string, description *string, permissionCodes []string) bool {
	*code = strings.TrimSpace(*code)
	*name = strings.TrimSpace(*name)
	if *code == "" || len(*code) > 50 || !roleCodePattern.MatchString(*code) || *name == "" || len(*name) > 100 || (description != nil && len(*description) > 500) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid role fields"})
		return false
	}
	seen := map[string]struct{}{}
	for _, permissionCode := range permissionCodes {
		if strings.TrimSpace(permissionCode) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid permission code"})
			return false
		}
		if _, exists := seen[permissionCode]; exists {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "permission_codes must be unique"})
			return false
		}
		seen[permissionCode] = struct{}{}
	}
	sort.Strings(permissionCodes)
	return true
}

func roleResponse(role orbit.Role, permissionCodes []string) roleResp {
	if permissionCodes == nil {
		permissionCodes = []string{}
	}
	return roleResp{Id: role.Id, Code: role.Code, Name: role.Name, Description: role.Description, CreatedAt: formatTime(role.CreatedAt), UpdatedAt: formatTime(role.UpdatedAt), PermissionCodes: permissionCodes}
}

func permissionResponse(permission orbit.Permission) permissionResp {
	return permissionResp{Id: permission.Id, Code: permission.Code, Name: permission.Name, Description: permission.Description}
}
