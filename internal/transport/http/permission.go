package transporthttp

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
)

func (s Server) requirePermission(c *gin.Context, permission string) (currentUserResp, bool) {
	user, ok := s.currentUser(c)
	if !ok {
		return currentUserResp{}, false
	}
	permissions, err := s.userRepository.UserPermissions(c.Request.Context(), user.Id)
	if err != nil {
		s.logger.Error("load current user permissions failed", "user_id", user.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to load user permissions"})
		return currentUserResp{}, false
	}
	for _, item := range permissions {
		if item == permission {
			return currentUserResp{User: user, Permissions: permissions}, true
		}
	}
	c.JSON(http.StatusForbidden, gin.H{"detail": "Permission denied"})
	return currentUserResp{}, false
}

type currentUserResp struct {
	User        model.User
	Permissions []string
}
