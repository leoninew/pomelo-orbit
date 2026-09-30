package pipelinerunhandler

import (
	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
)

func (h Handler) StreamStageLog(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	subscription, err := h.service.OpenStageLogStream(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("run_id"), c.Param("stage_run_id"), c.Query("cursor"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteLogStream(c, subscription)
}
