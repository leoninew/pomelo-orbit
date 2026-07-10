package routes

import (
	"github.com/gin-gonic/gin"

	projecthandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/project"
)

func (r Router) registerProject(engine *gin.Engine) {
	handler := projecthandler.New(r.logger, r.deps.ProjectService, r.deps.Authenticator)
	engine.GET("/api/project", handler.ListProjects)
	engine.POST("/api/project", handler.CreateProject)
	engine.GET("/api/project/:project_id", handler.GetProject)
	engine.PUT("/api/project/:project_id", handler.UpdateProject)
	engine.POST("/api/project/:project_id/deprecate", handler.DeprecateProject)
	engine.GET("/api/project/:project_id/member", handler.ListProjectMembers)
	engine.POST("/api/project/:project_id/member", handler.AddProjectMember)
	engine.DELETE("/api/project/:project_id/member/:user_id", handler.RemoveProjectMember)
}
