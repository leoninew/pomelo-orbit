package cihandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
)

func (h Handler) ListRepositoryRuns(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRepositoryRuns(c.Request.Context(), current.Id, c.Param("repository_id"), page, perPage)
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.PipelineRunResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, pipelineRunResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.PipelineRunPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) TriggerRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineRunTriggerReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.TriggerRepository(c.Request.Context(), current.Id, cidto.PipelineRunTriggerInput{RepositoryId: c.Param("repository_id"), TemplateId: req.TemplateId, TriggerRef: req.TriggerRef, Variables: req.Variables})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) ListPipelineRuns(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListPipelineRuns(c.Request.Context(), current.Id, cidto.PipelineRunListInput{ProjectId: c.Request.URL.Query().Get("project_id"), RepositoryId: c.Request.URL.Query().Get("repository_id"), TemplateId: c.Request.URL.Query().Get("template_id"), DateFrom: c.Request.URL.Query().Get("date_from"), DateTo: c.Request.URL.Query().Get("date_to"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.PipelineRunResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, pipelineRunResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.PipelineRunPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) GetPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineRunForUser(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ListPipelineRunArtifacts(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListPipelineRunArtifacts(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	responses := make([]pomeloorbit.ArtifactResp, 0, len(items))
	for _, item := range items {
		responses = append(responses, artifactResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.PipelineRunArtifactListResp{Items: transportresponse.Ptrs(responses)})
}

func (h Handler) GetPipelineStageLog(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	result, err := h.service.PipelineStageLog(c.Request.Context(), current.Id, c.Param("run_id"), c.Param("stage_run_id"), binding.QueryInt(c.Request.URL.Query().Get("offset"), 0))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.PipelineStageLogResp{Logs: result.Logs, Offset: int32(result.Offset), IsComplete: result.IsComplete})
}

func (h Handler) CancelPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineRunCancelReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CancelPipelineRun(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) RetryPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineRunRetryReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.RetryPipelineRun(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineRunResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}
