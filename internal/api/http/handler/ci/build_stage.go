package cihandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
)

func (h Handler) ListBuildStages(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListBuildStages(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.BuildStageResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, buildStageDetailResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.BuildStagePaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	stage, err := h.service.CreateBuildStage(c.Request.Context(), current.Id, cidto.BuildStageCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: serviceArtifacts(req.Artifacts), Description: req.Description})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	stage, err := h.service.BuildStageForUser(c.Request.Context(), current.Id, c.Param("stage_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var artifacts *[]cidto.ArtifactConfig
	if req.Artifacts != nil {
		items := serviceArtifacts(req.Artifacts.Items)
		artifacts = &items
	}
	stage, err := h.service.UpdateBuildStage(c.Request.Context(), current.Id, c.Param("stage_id"), cidto.BuildStageUpdateInput{Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: artifacts, Description: req.Description})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteBuildStage(c.Request.Context(), current.Id, c.Param("stage_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) DuplicateBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageDuplicateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	stage, err := h.service.DuplicateBuildStage(c.Request.Context(), current.Id, c.Param("stage_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func buildStageDetailResponse(item cidto.BuildStageDetail) pomeloorbit.BuildStageResp {
	return pomeloorbit.BuildStageResp{Id: item.Id, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: transportresponse.Ptrs(artifactConfigsResponse(item.Artifacts)), Description: item.Description, Version: int32(item.Version), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func serviceArtifacts(items []*pomeloorbit.ArtifactConfigReq) []cidto.ArtifactConfig {
	resp := make([]cidto.ArtifactConfig, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, cidto.ArtifactConfig{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}
