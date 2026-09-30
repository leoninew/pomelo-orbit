package deploymenthandler

import (
	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
)

func (h Handler) StreamDeploymentLog(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	subscription, err := h.service.OpenDeploymentLogStream(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("deployment_id"), c.Query("cursor"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteLogStream(c, subscription)
}

func (h Handler) StreamDeploymentContainerLog(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	subscription, err := h.service.OpenDeploymentContainerLogStream(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("deployment_id"), c.Query("cursor"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteLogStream(c, subscription)
}

func (h Handler) StreamApplicationLog(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	subscription, err := h.service.OpenApplicationLogStream(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("app_id"), c.Query("service_id"), c.Query("component"), c.Query("cursor"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteLogStream(c, subscription)
}
