package pipelinehandler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	pipelinev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/pipeline"
)

func (h Handler) ListPipelines(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page, perPage := binding.QueryInt(c.Query("page"), 1), binding.QueryInt(c.Query("per_page"), 20)
	items, err := h.service.ListPipelines(c.Request.Context(), current.Id, c.Query("project_id"), c.Query("kind"), page, perPage, c.Query("search"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := make([]*pipelinev1.PipelineResp, 0, len(items.Items))
	for _, item := range items.Items {
		response = append(response, pipelineResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pipelinev1.PipelinePaginatedResp{Items: response, Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}
func (h Handler) CreatePipeline(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineCreateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CreatePipeline(c.Request.Context(), current.Id, pipelinedto.PipelineCreateInput{ProjectId: c.Query("project_id"), Kind: req.Kind, Name: req.Name, Description: req.Description, VariableDeclarations: variableRequestMaps(req.VariableDeclarations)})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, response)
}
func (h Handler) GetPipeline(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineForUser(c.Request.Context(), current.Id, c.Param("pipeline_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}
func (h Handler) UpdatePipeline(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineUpdateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var variables *[]map[string]any
	if req.VariableDeclarations != nil {
		values := variableRequestMaps(req.VariableDeclarations.Items)
		variables = &values
	}
	detail, err := h.service.UpdatePipeline(c.Request.Context(), current.Id, c.Param("pipeline_id"), pipelinedto.PipelineUpdateInput{Name: req.Name, Description: req.Description, VariableDeclarations: variables, ApplicationId: req.ApplicationId})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}
func (h Handler) DeletePipeline(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeletePipeline(c.Request.Context(), current.Id, c.Param("pipeline_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h Handler) InstantiatePipeline(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineInstantiateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	bindings := make([]pipelinedto.PipelineArtifactBinding, 0, len(req.ArtifactBindings))
	for _, item := range req.ArtifactBindings {
		if item != nil {
			bindings = append(bindings, pipelinedto.PipelineArtifactBinding{StageId: item.StageId, ArtifactName: item.ArtifactName, ComponentName: item.ComponentName})
		}
	}
	detail, err := h.service.InstantiatePipeline(c.Request.Context(), current.Id, c.Param("pipeline_id"), pipelinedto.PipelineInstantiateInput{Name: req.Name, ApplicationId: req.ApplicationId, RepositoryId: req.RepositoryId, VersionForkStrategy: req.VersionForkStrategy, FixedVersionId: req.FixedVersionId, ArtifactBindings: bindings})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, response)
}
func (h Handler) ImportPipelineStage(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageImportReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.ImportPipelineStage(c.Request.Context(), current.Id, c.Param("pipeline_id"), pipelinedto.PipelineStageImportInput{SourceTemplateStageId: req.SourceTemplateStageId, Name: req.Name, Description: req.Description, DependsOn: req.DependsOn, SortOrder: int(req.SortOrder)})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, response)
}
func (h Handler) UpdatePipelineStageNode(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pipelinev1.PipelineStageNodeUpdateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var dependsOn *[]string
	if req.DependsOn != nil {
		values := req.DependsOn.Items
		dependsOn = &values
	}
	detail, err := h.service.UpdatePipelineStageNode(c.Request.Context(), current.Id, c.Param("pipeline_id"), c.Param("stage_id"), pipelinedto.PipelineStageNodeUpdateInput{Name: req.Name, Image: req.Image, Script: req.Script, DependsOn: dependsOn, SortOrder: int32PtrToInt(req.SortOrder), Description: req.Description})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}
func (h Handler) DeletePipelineStageNode(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.DeletePipelineStageNode(c.Request.Context(), current.Id, c.Param("pipeline_id"), c.Param("stage_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}
func (h Handler) GetPipelineSnapshot(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineSnapshotForUser(c.Request.Context(), current.Id, c.Param("snapshot_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := pipelineSnapshotResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}
