package pipelinehandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pipelinev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
)

func (h Handler) ListPipelineStages(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListPipelineStages(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := make([]pipelinev1.PipelineStageResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, pipelineStageDetailResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pipelinev1.PipelineStagePaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreatePipelineStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	stage, err := h.service.CreatePipelineStage(c.Request.Context(), current.Id, pipelinedto.PipelineStageCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: serviceArtifacts(req.Artifacts), BuildVersionBinding: buildVersionBindingInput(req.BuildVersionBinding), Description: req.Description})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := pipelineStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetPipelineStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	stage, err := h.service.PipelineStageForUser(c.Request.Context(), current.Id, c.Param("stage_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := pipelineStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdatePipelineStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var artifacts *[]pipelinedto.ArtifactConfig
	if req.Artifacts != nil {
		items := serviceArtifacts(req.Artifacts.Items)
		artifacts = &items
	}
	stage, err := h.service.UpdatePipelineStage(c.Request.Context(), current.Id, c.Param("stage_id"), pipelinedto.PipelineStageUpdateInput{Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: artifacts, BuildVersionBinding: buildVersionBindingInput(req.BuildVersionBinding), ClearBuildVersionBinding: req.ClearBuildVersionBinding != nil && *req.ClearBuildVersionBinding, Description: req.Description})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := pipelineStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeletePipelineStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeletePipelineStage(c.Request.Context(), current.Id, c.Param("stage_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) DuplicatePipelineStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageDuplicateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	stage, err := h.service.DuplicatePipelineStage(c.Request.Context(), current.Id, c.Param("stage_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := pipelineStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}
