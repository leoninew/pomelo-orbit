package routes

import (
	"github.com/gin-gonic/gin"

	userhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/user"
)

func (r Router) registerUser(engine *gin.Engine) {
	handler := userhandler.New(r.logger, r.deps.UserService, r.deps.Authenticator)
	engine.GET("/api/user", handler.ListUsers)
	engine.POST("/api/user", handler.CreateUser)
	engine.GET("/api/user/:user_id", handler.GetUser)
	engine.PUT("/api/user/:user_id", handler.UpdateUser)
	engine.PUT("/api/user/:user_id/role", handler.UpdateUserRoles)
	engine.POST("/api/user/:user_id/disable", handler.DisableUser)
	engine.POST("/api/user/:user_id/enable", handler.EnableUser)
	engine.DELETE("/api/user/:user_id", handler.DeleteUser)
}
