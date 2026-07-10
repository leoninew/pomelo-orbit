package cihandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
)

func (h Handler) ListPipelineTemplates(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListPipelineTemplates(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.PipelineTemplateResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, pipelineTemplateResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.PipelineTemplatePaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreatePipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineTemplateCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CreatePipelineTemplate(c.Request.Context(), current.Id, cidto.PipelineTemplateCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Description: req.Description, VariableDeclarations: variableDeclarationRequestMaps(req.VariableDeclarations)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineTemplateResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetPipelineTemplate(c *gin.Context) {
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
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdatePipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineTemplateUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var orchestration *[]cidto.StageOrchestration
	if req.Orchestration != nil {
		items := serviceOrchestration(req.Orchestration.Items)
		orchestration = &items
	}
	var variableDeclarations *[]map[string]any
	if req.VariableDeclarations != nil {
		items := variableDeclarationRequestMaps(req.VariableDeclarations.Items)
		variableDeclarations = &items
	}
	detail, err := h.service.UpdatePipelineTemplate(c.Request.Context(), current.Id, c.Param("template_id"), cidto.PipelineTemplateUpdateInput{Name: req.Name, Description: req.Description, Orchestration: orchestration, VariableDeclarations: variableDeclarations})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineTemplateResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeletePipelineTemplate(c *gin.Context) {
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

func (h Handler) DuplicatePipelineTemplate(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineTemplateDuplicateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.DuplicatePipelineTemplate(c.Request.Context(), current.Id, c.Param("template_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineTemplateResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) ResolvePipelineTemplateVariables(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.TemplateVariableResolveReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	variables, err := h.service.ResolvePipelineTemplateVariables(c.Request.Context(), current.Id, cidto.PipelineTemplateResolveInput{ProjectId: c.Request.URL.Query().Get("project_id"), Orchestration: serviceOrchestration(req.Orchestration), VariableDeclarations: variableDeclarationRequestMaps(req.VariableDeclarations)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.TemplateVariableResolveResp{Items: transportresponse.Ptrs(variableDeclarationResponses(variables))})
}

func pipelineTemplateResponse(detail cidto.PipelineTemplateDetail) pomeloorbit.PipelineTemplateResp {
	item := detail.Template
	return pomeloorbit.PipelineTemplateResp{Id: item.Id, Name: item.Name, Description: item.Description, Orchestration: transportresponse.Ptrs(orchestrationResponse(detail.Orchestration)), Stages: transportresponse.Ptrs(buildStageDetailsResponse(detail.Stages)), VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)), Version: int32(item.Version), CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func orchestrationResponse(items []cidto.StageOrchestration) []pomeloorbit.StageOrchestrationResp {
	resp := make([]pomeloorbit.StageOrchestrationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.StageOrchestrationResp{StageId: item.StageId, StageName: item.StageName, StageVersion: int32(item.StageVersion), DependsOn: item.DependsOn, SortOrder: int32(item.SortOrder)})
	}
	return resp
}

func serviceOrchestration(items []*pomeloorbit.StageOrchestrationReq) []cidto.StageOrchestration {
	resp := make([]cidto.StageOrchestration, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, cidto.StageOrchestration{StageId: item.StageId, StageName: item.StageName, StageVersion: int(item.StageVersion), DependsOn: item.DependsOn, SortOrder: int(item.SortOrder)})
	}
	return resp
}

func buildStageDetailsResponse(items []cidto.BuildStageDetail) []pomeloorbit.BuildStageResp {
	resp := make([]pomeloorbit.BuildStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, buildStageDetailResponse(item))
	}
	return resp
}

func artifactConfigsResponse(items []cidto.ArtifactConfig) []pomeloorbit.ArtifactConfigResp {
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
