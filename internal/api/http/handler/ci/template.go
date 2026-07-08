package cihandler

import (
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"
	"net/http"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci"
)

func (h Handler) RegisterTemplateRoutes(r router) {
	r.GET("/api/ci/template", h.listPipelineTemplates)
	r.POST("/api/ci/template", h.createPipelineTemplate)
	r.POST("/api/ci/template/resolve-variables", h.resolvePipelineTemplateVariables)
	r.GET("/api/ci/template/:template_id", h.getPipelineTemplate)
	r.PUT("/api/ci/template/:template_id", h.updatePipelineTemplate)
	r.DELETE("/api/ci/template/:template_id", h.deletePipelineTemplate)
	r.POST("/api/ci/template/:template_id/duplicate", h.duplicatePipelineTemplate)
}

func (h Handler) listPipelineTemplates(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListPipelineTemplates(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, pipelineTemplateResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.PipelineTemplatePaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) createPipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineTemplateCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.CreatePipelineTemplate(c.Request.Context(), current.Id, cisvc.PipelineTemplateCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Description: req.Description, VariableDeclarations: variableDeclarationRequestMaps(req.VariableDeclarations)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineTemplateResponse(detail)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getPipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.PipelineTemplateForUser(c.Request.Context(), current.Id, c.Param("template_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineTemplateResponse(detail)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updatePipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineTemplateUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	var orchestration *[]cisvc.StageOrchestration
	if req.Orchestration != nil {
		items := serviceOrchestration(req.Orchestration.Items)
		orchestration = &items
	}
	var variableDeclarations *[]map[string]any
	if req.VariableDeclarations != nil {
		items := variableDeclarationRequestMaps(req.VariableDeclarations.Items)
		variableDeclarations = &items
	}
	detail, err := h.service.UpdatePipelineTemplate(c.Request.Context(), current.Id, c.Param("template_id"), cisvc.PipelineTemplateUpdateInput{Name: req.Name, Description: req.Description, Orchestration: orchestration, VariableDeclarations: variableDeclarations})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineTemplateResponse(detail)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deletePipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeletePipelineTemplate(c.Request.Context(), current.Id, c.Param("template_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) duplicatePipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineTemplateDuplicateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.DuplicatePipelineTemplate(c.Request.Context(), current.Id, c.Param("template_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineTemplateResponse(detail)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) resolvePipelineTemplateVariables(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.TemplateVariableResolveReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	variables, err := h.service.ResolvePipelineTemplateVariables(c.Request.Context(), current.Id, cisvc.PipelineTemplateResolveInput{ProjectId: c.Request.URL.Query().Get("project_id"), Orchestration: serviceOrchestration(req.Orchestration), VariableDeclarations: variableDeclarationRequestMaps(req.VariableDeclarations)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.TemplateVariableResolveResp{Items: transportresponse.Ptrs(variableDeclarationResponses(variables))}})
}

func pipelineTemplateResponse(detail cisvc.PipelineTemplateDetail) pomeloorbit.PipelineTemplateResp {
	item := detail.Template
	return pomeloorbit.PipelineTemplateResp{Id: item.Id, Name: item.Name, Description: item.Description, Orchestration: transportresponse.Ptrs(orchestrationResponse(detail.Orchestration)), Stages: transportresponse.Ptrs(buildStageDetailsResponse(detail.Stages)), VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)), Version: int32(item.Version), CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func orchestrationResponse(items []cisvc.StageOrchestration) []pomeloorbit.StageOrchestrationResp {
	resp := make([]pomeloorbit.StageOrchestrationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.StageOrchestrationResp{StageId: item.StageId, StageName: item.StageName, StageVersion: int32(item.StageVersion), DependsOn: item.DependsOn, SortOrder: int32(item.SortOrder)})
	}
	return resp
}

func serviceOrchestration(items []*pomeloorbit.StageOrchestrationReq) []cisvc.StageOrchestration {
	resp := make([]cisvc.StageOrchestration, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, cisvc.StageOrchestration{StageId: item.StageId, StageName: item.StageName, StageVersion: int(item.StageVersion), DependsOn: item.DependsOn, SortOrder: int(item.SortOrder)})
	}
	return resp
}

func buildStageDetailsResponse(items []cisvc.BuildStageDetail) []pomeloorbit.BuildStageResp {
	resp := make([]pomeloorbit.BuildStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, buildStageDetailResponse(item))
	}
	return resp
}

func artifactConfigsResponse(items []cisvc.ArtifactConfig) []pomeloorbit.ArtifactConfigResp {
	if items == nil {
		return nil
	}
	resp := make([]pomeloorbit.ArtifactConfigResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.ArtifactConfigResp{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}

func variableDeclarationResponses(items []map[string]any) []pomeloorbit.VariableDeclarationResp {
	resp := make([]pomeloorbit.VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, variableDeclarationResponse(item))
	}
	return resp
}

func variableDeclarationResponse(item map[string]any) pomeloorbit.VariableDeclarationResp {
	return pomeloorbit.VariableDeclarationResp{Name: stringFromMap(item, "name"), Description: stringFromMap(item, "description"), Default: transportresponse.ProtoValue(item["default"]), Value: transportresponse.ProtoValue(item["value"]), Secret: boolFromMap(item, "secret"), Source: stringFromMap(item, "source"), Editable: boolFromMap(item, "editable")}
}

func variableDeclarationRequestMaps(items []*pomeloorbit.VariableDeclarationReq) []map[string]any {
	resp := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, variableDeclarationRequestMap(item))
	}
	return resp
}

func variableDeclarationRequestMap(item *pomeloorbit.VariableDeclarationReq) map[string]any {
	return map[string]any{"name": item.Name, "description": item.Description, "default": transportresponse.NativeValue(item.Default), "value": transportresponse.NativeValue(item.Value), "secret": item.Secret, "source": item.Source, "editable": item.Editable}
}

func stringFromMap(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return value
}

func boolFromMap(item map[string]any, key string) bool {
	value, _ := item[key].(bool)
	return value
}
