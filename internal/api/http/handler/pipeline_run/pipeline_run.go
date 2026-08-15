package pipelinerunhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	pipelinerunv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/pipeline_run"
)

func (h Handler) TriggerPipeline(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinerunv1.PipelineRunTriggerReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.TriggerPipeline(c.Request.Context(), current.Id, c.Param("pipeline_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, response)
}
func (h Handler) ListPipelineRuns(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page, perPage := binding.QueryInt(c.Query("page"), 1), binding.QueryInt(c.Query("per_page"), 20)
	items, err := h.service.ListPipelineRuns(c.Request.Context(), current.Id, pipelinerundto.PipelineRunListInput{ProjectId: c.Query("project_id"), RepositoryId: c.Query("repository_id"), PipelineId: c.Query("pipeline_id"), DateFrom: c.Query("date_from"), DateTo: c.Query("date_to"), Page: page, PerPage: perPage})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := make([]*pipelinerunv1.PipelineRunResp, 0, len(items.Items))
	for _, item := range items.Items {
		response = append(response, pipelineRunResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pipelinerunv1.PipelineRunPaginatedResp{Items: response, Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}
func (h Handler) GetPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineRunForUser(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}
func (h Handler) DeletePipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeletePipelineRun(c.Request.Context(), current.Id, c.Param("run_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h Handler) ListPipelineRunArtifacts(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListPipelineRunArtifacts(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := make([]pipelinerunv1.ArtifactResp, 0, len(items))
	for _, item := range items {
		response = append(response, artifactResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pipelinerunv1.PipelineRunArtifactListResp{Items: transportresponse.Ptrs(response)})
}
func (h Handler) GetPipelineStageLog(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineStageLog(c.Request.Context(), current.Id, c.Param("run_id"), c.Param("stage_run_id"), binding.QueryInt(c.Query("offset"), 0))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pipelinerunv1.PipelineStageLogResp{Logs: detail.Logs, Offset: int32(detail.Offset), IsComplete: detail.IsComplete})
}
func (h Handler) CancelPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinerunv1.PipelineRunCancelReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CancelPipelineRun(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}
func (h Handler) RetryPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinerunv1.PipelineRunRetryReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.RetryPipelineRun(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, response)
}
