package httpserver

import (
	"net/http"

	"backend/internal/orbit"
)

func (s Server) requirePermission(w http.ResponseWriter, r *http.Request, permission string) (currentUserResp, bool) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return currentUserResp{}, false
	}
	permissions, err := s.store.UserPermissions(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load current user permissions failed", "user_id", user.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user permissions"})
		return currentUserResp{}, false
	}
	for _, item := range permissions {
		if item == permission {
			return currentUserResp{User: user, Permissions: permissions}, true
		}
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
	return currentUserResp{}, false
}

type currentUserResp struct {
	User        orbit.User
	Permissions []string
}
