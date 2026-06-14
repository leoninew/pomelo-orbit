package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"backend/internal/orbit"

	"golang.org/x/crypto/bcrypt"
)

type userRoleResp struct {
	Id              string   `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	PermissionCodes []string `json:"permission_codes"`
}

type userListResp struct {
	Id          string         `json:"id"`
	Username    string         `json:"username"`
	Email       *string        `json:"email"`
	AuthSource  string         `json:"auth_source"`
	CreatedAt   string         `json:"created_at"`
	LastLoginAt *string        `json:"last_login_at"`
	Status      string         `json:"status"`
	UpdatedAt   string         `json:"updated_at"`
	RoleItems   []userRoleResp `json:"role_items"`
}

type userResp struct {
	Id          string         `json:"id"`
	Username    string         `json:"username"`
	Email       *string        `json:"email"`
	AuthSource  string         `json:"auth_source"`
	CreatedAt   string         `json:"created_at"`
	LastLoginAt *string        `json:"last_login_at"`
	Roles       []string       `json:"roles"`
	Permissions []string       `json:"permissions"`
	Status      string         `json:"status"`
	UpdatedAt   string         `json:"updated_at"`
	RoleItems   []userRoleResp `json:"role_items"`
}

type userCreateReq struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Email    *string `json:"email"`
}

type userUpdateReq struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	Status   *string `json:"status"`
}

type userRoleUpdateReq struct {
	RoleIds []string `json:"role_ids"`
}

func (s Server) registerUserRoutes(r chiRouter) {
	r.Get("/api/user", s.listUsers)
	r.Post("/api/user", s.createUser)
	r.Get("/api/user/{user_id}", s.getUser)
	r.Put("/api/user/{user_id}", s.updateUser)
	r.Put("/api/user/{user_id}/role", s.updateUserRoles)
	r.Post("/api/user/{user_id}/disable", s.disableUser)
	r.Post("/api/user/{user_id}/enable", s.enableUser)
	r.Delete("/api/user/{user_id}", s.deleteUser)
}

func (s Server) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "user:read"); !ok {
		return
	}
	page, perPage := pageParams(r)
	users, err := s.store.ListUsers(r.Context(), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list users failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list users"})
		return
	}
	userIds := make([]string, 0, len(users.Items))
	for _, user := range users.Items {
		userIds = append(userIds, user.Id)
	}
	rolesByUserId, err := s.store.UserRolesByUserIds(r.Context(), userIds)
	if err != nil {
		s.logger.Error("load users roles failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user roles"})
		return
	}
	resp := mapPage(users, func(user orbit.User) userListResp {
		return userListResponse(user, rolesByUserId[user.Id])
	})
	writeJSON(w, http.StatusOK, newPaginatedResp(resp))
}

func (s Server) createUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "user:write"); !ok {
		return
	}
	var req userCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if email == "" {
			req.Email = nil
		} else {
			req.Email = &email
		}
	}
	if req.Username == "" || len(req.Username) > 50 || len(req.Password) < 6 || len(req.Password) > 255 || (req.Email != nil && len(*req.Email) > 255) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid user fields"})
		return
	}
	if _, err := s.store.UserByUsername(r.Context(), req.Username); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"detail": "Username " + req.Username + " already exists"})
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check username failed", "username", req.Username, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check username"})
		return
	}
	if req.Email != nil {
		if _, err := s.store.UserByEmail(r.Context(), *req.Email); err == nil {
			writeJSON(w, http.StatusConflict, map[string]string{"detail": "Email " + *req.Email + " already exists"})
			return
		} else if !errors.Is(err, sql.ErrNoRows) {
			s.logger.Error("check email failed", "email", *req.Email, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check email"})
			return
		}
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("hash user password failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to hash password"})
		return
	}
	now := time.Now().UTC()
	user := orbit.User{Id: orbit.NewId(), Username: req.Username, PasswordHash: string(passwordHash), Status: "enabled", OAuthProvider: "", OAuthProviderId: "", Email: req.Email, AuthSource: "password", CreatedAt: now, UpdatedAt: now}
	if err := s.store.CreateUser(r.Context(), user); err != nil {
		s.logger.Error("create user failed", "username", req.Username, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create user"})
		return
	}
	resp, ok := s.userDetailResp(w, r, user)
	if !ok {
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s Server) getUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "user:read"); !ok {
		return
	}
	user, ok := s.loadUserFromPath(w, r)
	if !ok {
		return
	}
	resp, ok := s.userDetailResp(w, r, user)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) updateUser(w http.ResponseWriter, r *http.Request) {
	current, ok := s.requirePermission(w, r, "user:write")
	if !ok {
		return
	}
	user, ok := s.loadUserFromPath(w, r)
	if !ok {
		return
	}
	var req userUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)
		if username == "" || len(username) > 50 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid user fields"})
			return
		}
		user.Username = username
	}
	if req.Password != nil {
		password := *req.Password
		if password != "" {
			if len(password) < 6 || len(password) > 255 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid user fields"})
				return
			}
			if !s.canResetPassword(r, current, user.Id) {
				writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
				return
			}
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				s.logger.Error("hash user password failed", "error", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to hash password"})
				return
			}
			user.PasswordHash = string(passwordHash)
		}
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != "enabled" && status != "disabled" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid user status"})
			return
		}
		if current.User.Id == user.Id && status == "disabled" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cannot disable current user"})
			return
		}
		user.Status = status
	}
	if err := s.store.UpdateUser(r.Context(), user); err != nil {
		s.logger.Error("update user failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update user"})
		return
	}
	updated, err := s.store.UserById(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load updated user failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user"})
		return
	}
	resp, ok := s.userDetailResp(w, r, updated)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) updateUserRoles(w http.ResponseWriter, r *http.Request) {
	current, ok := s.requirePermission(w, r, "user:write")
	if !ok {
		return
	}
	if !hasPermission(current.Permissions, "role:write") {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
		return
	}
	user, ok := s.loadUserFromPath(w, r)
	if !ok {
		return
	}
	var req userRoleUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	seen := map[string]struct{}{}
	for _, roleId := range req.RoleIds {
		roleId = strings.TrimSpace(roleId)
		if roleId == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid role id"})
			return
		}
		if _, exists := seen[roleId]; exists {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "role_ids must be unique"})
			return
		}
		seen[roleId] = struct{}{}
		if _, err := s.store.RoleById(r.Context(), roleId); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Role " + roleId + " not found"})
				return
			}
			s.logger.Error("load role failed", "role_id", roleId, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load role"})
			return
		}
	}
	if err := s.store.SetUserRoles(r.Context(), user.Id, req.RoleIds); err != nil {
		s.logger.Error("update user roles failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update user roles"})
		return
	}
	updated, err := s.store.UserById(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load updated user failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user"})
		return
	}
	resp, ok := s.userDetailResp(w, r, updated)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) disableUser(w http.ResponseWriter, r *http.Request) {
	current, ok := s.requirePermission(w, r, "user:write")
	if !ok {
		return
	}
	user, ok := s.loadUserFromPath(w, r)
	if !ok {
		return
	}
	if current.User.Id == user.Id {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cannot disable current user"})
		return
	}
	if user.Status != "disabled" {
		if err := s.store.SetUserStatus(r.Context(), user.Id, "disabled"); err != nil {
			s.logger.Error("disable user failed", "user_id", user.Id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to disable user"})
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) enableUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePermission(w, r, "user:write"); !ok {
		return
	}
	user, ok := s.loadUserFromPath(w, r)
	if !ok {
		return
	}
	if user.Status != "enabled" {
		if err := s.store.SetUserStatus(r.Context(), user.Id, "enabled"); err != nil {
			s.logger.Error("enable user failed", "user_id", user.Id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enable user"})
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	current, ok := s.requirePermission(w, r, "user:write")
	if !ok {
		return
	}
	user, ok := s.loadUserFromPath(w, r)
	if !ok {
		return
	}
	if current.User.Id == user.Id {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cannot delete current user"})
		return
	}
	if err := s.store.DeleteUser(r.Context(), user.Id); err != nil {
		s.logger.Error("delete user failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete user"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) loadUserFromPath(w http.ResponseWriter, r *http.Request) (orbit.User, bool) {
	userId := urlParam(r, "user_id")
	user, err := s.store.UserById(r.Context(), userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "User " + userId + " not found"})
			return orbit.User{}, false
		}
		s.logger.Error("load user failed", "user_id", userId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user"})
		return orbit.User{}, false
	}
	return user, true
}

func (s Server) userDetailResp(w http.ResponseWriter, r *http.Request, user orbit.User) (userResp, bool) {
	roles, err := s.store.UserRoleDetails(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load user roles failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user roles"})
		return userResp{}, false
	}
	permissions, err := s.store.UserPermissions(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user permissions"})
		return userResp{}, false
	}
	roleIds := make([]string, 0, len(roles))
	roleCodes := make([]string, 0, len(roles))
	for _, role := range roles {
		roleIds = append(roleIds, role.Id)
		roleCodes = append(roleCodes, role.Code)
	}
	permissionsByRoleId, err := s.store.RolePermissionCodesByRoleIds(r.Context(), roleIds)
	if err != nil {
		s.logger.Error("load role permissions failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load role permissions"})
		return userResp{}, false
	}
	return userResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: formatTime(user.CreatedAt), LastLoginAt: formatOptionalTime(user.LastLoginAt), Roles: roleCodes, Permissions: permissions, Status: user.Status, UpdatedAt: formatTime(user.UpdatedAt), RoleItems: roleResponses(roles, permissionsByRoleId)}, true
}

func userListResponse(user orbit.User, roles []orbit.Role) userListResp {
	return userListResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: formatTime(user.CreatedAt), LastLoginAt: formatOptionalTime(user.LastLoginAt), Status: user.Status, UpdatedAt: formatTime(user.UpdatedAt), RoleItems: roleResponses(roles, nil)}
}

func roleResponses(roles []orbit.Role, permissionsByRoleId map[string][]string) []userRoleResp {
	if roles == nil {
		return []userRoleResp{}
	}
	items := make([]userRoleResp, 0, len(roles))
	for _, role := range roles {
		permissionCodes := []string{}
		if permissionsByRoleId != nil {
			permissionCodes = permissionsByRoleId[role.Id]
			if permissionCodes == nil {
				permissionCodes = []string{}
			}
		}
		items = append(items, userRoleResp{Id: role.Id, Code: role.Code, Name: role.Name, PermissionCodes: permissionCodes})
	}
	return items
}

func hasPermission(permissions []string, permission string) bool {
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func (s Server) canResetPassword(r *http.Request, current currentUserResp, targetUserId string) bool {
	if current.User.Id == targetUserId || hasPermission(current.Permissions, "role:write") {
		return true
	}
	targetPermissions, err := s.store.UserPermissions(r.Context(), targetUserId)
	if err != nil {
		s.logger.Warn("load target user permissions failed", "user_id", targetUserId, "error", err)
		return false
	}
	return !hasPermission(targetPermissions, "role:write")
}
