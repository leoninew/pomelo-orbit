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
