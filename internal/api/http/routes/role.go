package routes

import (
	"github.com/gin-gonic/gin"

	rolehandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/role"
)

func (r Router) registerRole(engine *gin.Engine) {
	handler := rolehandler.New(r.logger, r.deps.RoleService, r.deps.Authenticator)
	engine.GET("/api/role", handler.ListRoles)
	engine.POST("/api/role", handler.CreateRole)
	engine.GET("/api/role/permission", handler.ListPermissions)
	engine.GET("/api/role/:role_id", handler.GetRole)
	engine.PUT("/api/role/:role_id", handler.UpdateRole)
	engine.DELETE("/api/role/:role_id", handler.DeleteRole)
}
