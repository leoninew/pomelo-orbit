package cihandler

import (
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/codec"
	"github.com/gin-gonic/gin"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/ci"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

func (h Handler) RegisterPipelineRunRoutes(r router) {
	r.GET("/api/ci/repository/:repository_id/run", h.listRepositoryRuns)
	r.POST("/api/ci/repository/:repository_id/trigger", h.triggerRepository)
	r.GET("/api/ci/run", h.listPipelineRuns)
	r.GET("/api/ci/run/:run_id", h.getPipelineRun)
	r.GET("/api/ci/run/:run_id/artifacts", h.listPipelineRunArtifacts)
	r.GET("/api/ci/run/:run_id/stages/:stage_run_id/log", h.getPipelineStageLog)
	r.POST("/api/ci/run/:run_id/cancel", h.cancelPipelineRun)
	r.POST("/api/ci/run/:run_id/retry", h.retryPipelineRun)
}

func (h Handler) listRepositoryRuns(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRepositoryRuns(c.Request.Context(), current.Id, c.Param("repository_id"), page, perPage)
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, pipelineRunResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.PipelineRunPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) triggerRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineRunTriggerReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.TriggerRepository(c.Request.Context(), current.Id, cisvc.PipelineRunTriggerInput{RepositoryId: c.Param("repository_id"), TemplateId: req.TemplateId, TriggerRef: req.TriggerRef, Variables: req.Variables})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineRunResponse(detail)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) listPipelineRuns(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListPipelineRuns(c.Request.Context(), current.Id, cisvc.PipelineRunListInput{ProjectId: c.Request.URL.Query().Get("project_id"), RepositoryId: c.Request.URL.Query().Get("repository_id"), TemplateId: c.Request.URL.Query().Get("template_id"), DateFrom: c.Request.URL.Query().Get("date_from"), DateTo: c.Request.URL.Query().Get("date_to"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, pipelineRunResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.PipelineRunPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) getPipelineRun(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) listPipelineRunArtifacts(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.PipelineRunArtifactListResp{Items: transportresponse.Ptrs(responses)}})
}

func (h Handler) getPipelineStageLog(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	result, err := h.service.PipelineStageLog(c.Request.Context(), current.Id, c.Param("run_id"), c.Param("stage_run_id"), transportresponse.QueryInt(c.Request.URL.Query().Get("offset"), 0))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.PipelineStageLogResp{Logs: result.Logs, Offset: int32(result.Offset), IsComplete: result.IsComplete}})
}

func (h Handler) cancelPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineRunCancelReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.CancelPipelineRun(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineRunResponse(detail)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) retryPipelineRun(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.PipelineRunRetryReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.RetryPipelineRun(c.Request.Context(), current.Id, c.Param("run_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pipelineRunResponse(detail)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func pipelineRunResponse(detail cisvc.PipelineRunDetail) pomeloorbit.PipelineRunResp {
	item := detail.Run
	return pomeloorbit.PipelineRunResp{Id: item.Id, ProjectId: item.ProjectId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotId, TemplateId: item.TemplateId, TemplateName: item.TemplateName, TemplateVersion: int32(item.TemplateVersion), Trigger: item.Trigger, TriggerRef: item.TriggerRef, VariablesSnapshot: transportresponse.Ptrs(pipelineRunVariableDeclarationResponses(detail.VariablesSnapshot)), Status: item.Status, RetryOf: item.RetryOf, StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: transportresponse.FormatTime(item.CreatedAt), StageRuns: transportresponse.Ptrs(stageRunsResponse(detail.StageRuns))}
}

func pipelineRunVariableDeclarationResponses(items []model.VariableDeclaration) []pomeloorbit.VariableDeclarationResp {
	resp := make([]pomeloorbit.VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, pomeloorbit.VariableDeclarationResp{Name: item.Name, Description: item.Description, Default: transportresponse.ProtoValue(item.Default), Value: transportresponse.ProtoValue(item.Value), Secret: item.Secret, Source: item.Source, Editable: item.Editable})
	}
	return resp
}

func stageRunsResponse(items []model.StageRun) []pomeloorbit.StageRunResp {
	resp := make([]pomeloorbit.StageRunResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, stageRunResponse(item))
	}
	return resp
}

func stageRunResponse(item model.StageRun) pomeloorbit.StageRunResp {
	return pomeloorbit.StageRunResp{Id: item.Id, PipelineRunId: item.PipelineRunId, StageId: item.StageId, StageName: item.StageName, Status: item.Status, StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ExitCode: transportresponse.OptionalInt32(item.ExitCode), ErrorMessage: item.ErrorMessage}
}

func artifactResponse(item model.Artifact) pomeloorbit.ArtifactResp {
	return pomeloorbit.ArtifactResp{Id: item.Id, PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, TemplateId: item.TemplateId, TemplateName: item.TemplateName, StageName: item.StageName, Type: item.Type, Name: item.Name, Path: item.Path, CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
}
