package pipelinehandler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	pipelinev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/pipeline"
)

func (h Handler) ListPipelineStageTemplates(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page, perPage := binding.QueryInt(c.Query("page"), 1), binding.QueryInt(c.Query("per_page"), 20)
	items, err := h.service.ListPipelineStageTemplates(c.Request.Context(), current.Id, c.Query("project_id"), page, perPage, c.Query("search"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := make([]*pipelinev1.PipelineStageResp, 0, len(items.Items))
	for _, item := range items.Items {
		response = append(response, pipelineStageTemplateResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pipelinev1.PipelineStagePaginatedResp{Items: response, Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreatePipelineStageTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageCreateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CreatePipelineStageTemplate(c.Request.Context(), current.Id, pipelinedto.PipelineStageTemplateCreateInput{ProjectId: c.Query("project_id"), Name: req.Name, Image: req.Image, Script: req.Script, Description: req.Description, Artifacts: serviceArtifacts(req.Artifacts)})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusCreated, pipelineStageTemplateResponse(detail))
}

func (h Handler) GetPipelineStageTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineStageTemplateForUser(c.Request.Context(), current.Id, c.Param("stage_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, pipelineStageTemplateResponse(detail))
}

func (h Handler) UpdatePipelineStageTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageUpdateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var artifacts *[]pipelinedto.ArtifactConfig
	if req.Artifacts != nil {
		values := serviceArtifacts(req.Artifacts.Items)
		artifacts = &values
	}
	detail, err := h.service.UpdatePipelineStageTemplate(c.Request.Context(), current.Id, c.Param("stage_id"), pipelinedto.PipelineStageTemplateUpdateInput{Name: req.Name, Image: req.Image, Script: req.Script, Description: req.Description, Artifacts: artifacts})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, pipelineStageTemplateResponse(detail))
}

func (h Handler) DeletePipelineStageTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeletePipelineStageTemplate(c.Request.Context(), current.Id, c.Param("stage_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) PreviewPipelineStageTemplateUpdate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineStageTemplateUpdatePreview(c.Request.Context(), current.Id, c.Param("pipeline_id"), c.Param("stage_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, pipelineStageTemplateUpdatePreviewResponse(detail))
}

func (h Handler) ApplyPipelineStageTemplateUpdate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageTemplateUpdateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.ApplyPipelineStageTemplateUpdate(c.Request.Context(), current.Id, c.Param("pipeline_id"), c.Param("stage_id"), pipelinedto.PipelineStageTemplateApplyUpdateInput{ExpectedSourceTemplateStageVersion: int(req.ExpectedSourceTemplateStageVersion), TargetTemplateStageVersion: int(req.TargetTemplateStageVersion)})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, pipelineResponse(detail))
}
