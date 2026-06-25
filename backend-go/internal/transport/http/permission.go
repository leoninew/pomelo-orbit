package transporthttp

import (
	"net/http"

	"backend/internal/repository/model"
	transportresponse "backend/internal/transport/http/response"
)

func (s Server) requirePermission(w http.ResponseWriter, r *http.Request, permission string) (currentUserResp, bool) {
	user, ok := s.currentUser(w, r)
	if !ok {
		return currentUserResp{}, false
	}
	permissions, err := s.userRepository.UserPermissions(r.Context(), user.Id)
	if err != nil {
		s.logger.Error("load current user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.JSON(s.logger, w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user permissions"})
		return currentUserResp{}, false
	}
	for _, item := range permissions {
		if item == permission {
			return currentUserResp{User: user, Permissions: permissions}, true
		}
	}
	transportresponse.JSON(s.logger, w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
	return currentUserResp{}, false
}

type currentUserResp struct {
	User        model.User
	Permissions []string
}
