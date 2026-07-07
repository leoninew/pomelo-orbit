package cihandler

import (
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/codec"
	"github.com/gin-gonic/gin"
	"net/http"

	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/ci"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

func (h Handler) RegisterBuildStageRoutes(r router) {
	r.GET("/api/ci/build-stage", h.listBuildStages)
	r.POST("/api/ci/build-stage", h.createBuildStage)
	r.GET("/api/ci/build-stage/:stage_id", h.getBuildStage)
	r.PUT("/api/ci/build-stage/:stage_id", h.updateBuildStage)
	r.DELETE("/api/ci/build-stage/:stage_id", h.deleteBuildStage)
	r.POST("/api/ci/build-stage/:stage_id/duplicate", h.duplicateBuildStage)
}

func (h Handler) listBuildStages(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListBuildStages(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, buildStageDetailResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.BuildStagePaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) createBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	stage, err := h.service.CreateBuildStage(c.Request.Context(), current.Id, cisvc.BuildStageCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: serviceArtifacts(req.Artifacts), Description: req.Description})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getBuildStage(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	var artifacts *[]cisvc.ArtifactConfig
	if req.Artifacts != nil {
		items := serviceArtifacts(req.Artifacts.Items)
		artifacts = &items
	}
	stage, err := h.service.UpdateBuildStage(c.Request.Context(), current.Id, c.Param("stage_id"), cisvc.BuildStageUpdateInput{Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: artifacts, Description: req.Description})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteBuildStage(c *gin.Context) {
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

func (h Handler) duplicateBuildStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageDuplicateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	stage, err := h.service.DuplicateBuildStage(c.Request.Context(), current.Id, c.Param("stage_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func buildStageDetailResponse(item cisvc.BuildStageDetail) pomeloorbit.BuildStageResp {
	return pomeloorbit.BuildStageResp{Id: item.Id, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: transportresponse.Ptrs(artifactConfigsResponse(item.Artifacts)), Description: item.Description, Version: int32(item.Version), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func serviceArtifacts(items []*pomeloorbit.ArtifactConfigReq) []cisvc.ArtifactConfig {
	resp := make([]cisvc.ArtifactConfig, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, cisvc.ArtifactConfig{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}
