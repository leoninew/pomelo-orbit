package cihandler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
)

func (h Handler) GetPipelineSnapshot(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	snapshot, err := h.service.PipelineSnapshotForUser(c.Request.Context(), current.Id, c.Param("snapshot_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineSnapshotResponse(snapshot)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}
