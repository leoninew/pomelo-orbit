package transporthttp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"backend/internal/repository"
)

var pipelineTemplateCopyPattern = regexp.MustCompile(` copy( [0-9]+)?$`)
var templateVariablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9_]*)(?:\s*\|\s*(?:default|d)\s*\(\s*['\"]([^'\"]*)['\"]\s*\))?\s*\}\}`)

type pipelineTemplateResp struct {
	Id                   string                   `json:"id"`
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	Orchestration        []stageOrchestrationResp `json:"orchestration"`
	Stages               []buildStageResp         `json:"stages"`
	VariableDeclarations []map[string]any         `json:"variable_declarations"`
	Version              int                      `json:"version"`
	CreatedAt            string                   `json:"created_at"`
	UpdatedAt            string                   `json:"updated_at"`
}

type stageOrchestrationResp struct {
	StageId      string   `json:"stage_id"`
	StageName    string   `json:"stage_name"`
	StageVersion int      `json:"stage_version"`
	DependsOn    []string `json:"depends_on"`
	SortOrder    int      `json:"sort_order"`
}

type pipelineTemplateCreateReq struct {
	Name                 string           `json:"name"`
	Description          string           `json:"description"`
	VariableDeclarations []map[string]any `json:"variable_declarations"`
}

type templateVariableResolveReq struct {
	Orchestration        []stageOrchestrationResp `json:"orchestration"`
	VariableDeclarations []map[string]any         `json:"variable_declarations"`
}

func (s Server) listPipelineTemplates(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	page, perPage := pageParams(r)
	items, err := s.store.ListPipelineTemplates(r.Context(), projectId, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list pipeline templates failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list pipeline templates"})
		return
	}
	responses := make([]pipelineTemplateResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp, ok := s.pipelineTemplateResponse(w, r, item)
		if !ok {
			return
		}
		responses = append(responses, resp)
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(repository.Page[pipelineTemplateResp]{Items: responses, Total: items.Total, Page: items.Page, PerPage: items.PerPage}))
}

func (s Server) createPipelineTemplate(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	var req pipelineTemplateCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizePipelineTemplateCreateReq(w, &req) || !s.ensurePipelineTemplateNameAvailable(w, r, projectId, req.Name, "") {
		return
	}
	variables, ok := marshalPipelineTemplateVariables(w, sanitizePipelineTemplateVariables(req.VariableDeclarations))
	if !ok {
		return
	}
	template := repository.PipelineTemplate{Id: repository.NewId(), ProjectId: &projectId, Name: req.Name, Description: req.Description, VariableDeclarations: variables, Version: 1}
	if err := s.store.CreatePipelineTemplate(r.Context(), template); err != nil {
		s.logger.Error("create pipeline template failed", "template_name", template.Name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create pipeline template"})
		return
	}
	created, err := s.store.PipelineTemplate(r.Context(), template.Id)
	if err != nil {
		s.logger.Error("load created pipeline template failed", "template_id", template.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template"})
		return
	}
	s.writePipelineTemplateDetail(w, r, created, http.StatusCreated)
}

func (s Server) getPipelineTemplate(w http.ResponseWriter, r *http.Request) {
	template, ok := s.loadPipelineTemplateForCurrentUser(w, r)
	if !ok {
		return
	}
	s.writePipelineTemplateDetail(w, r, template, http.StatusOK)
}

func (s Server) updatePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	template, ok := s.loadPipelineTemplateForCurrentUser(w, r)
	if !ok {
		return
	}
	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	stages, orchestrationProvided, ok := s.applyPipelineTemplateUpdateReq(w, r, &template, req)
	if !ok {
		return
	}
	if orchestrationProvided {
		if err := s.store.UpdatePipelineTemplateWithStages(r.Context(), template, stages); err != nil {
			s.logger.Error("update pipeline template failed", "template_id", template.Id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update pipeline template"})
			return
		}
	} else if err := s.store.UpdatePipelineTemplate(r.Context(), template); err != nil {
		s.logger.Error("update pipeline template failed", "template_id", template.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update pipeline template"})
		return
	}
	updated, err := s.store.PipelineTemplate(r.Context(), template.Id)
	if err != nil {
		s.logger.Error("load updated pipeline template failed", "template_id", template.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template"})
		return
	}
	s.writePipelineTemplateDetail(w, r, updated, http.StatusOK)
}

func (s Server) deletePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	template, ok := s.loadPipelineTemplateForCurrentUser(w, r)
	if !ok {
		return
	}
	referenced, err := s.store.PipelineTemplateReferencedByWebhooks(r.Context(), template.Id)
	if err != nil {
		s.logger.Error("check pipeline template references failed", "template_id", template.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check pipeline template references"})
		return
	}
	if referenced {
		writeJSON(w, http.StatusConflict, map[string]string{"detail": "Template is referenced by webhooks, cannot delete"})
		return
	}
	if err := s.store.DeletePipelineTemplate(r.Context(), template.Id); err != nil {
		s.logger.Error("delete pipeline template failed", "template_id", template.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete pipeline template"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) duplicatePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	template, ok := s.loadPipelineTemplateForCurrentUser(w, r)
	if !ok {
		return
	}
	projectId := pipelineTemplateProjectId(template)
	name, ok := s.nextPipelineTemplateCopyName(w, r, projectId, template.Name)
	if !ok {
		return
	}
	existingStages, err := s.store.PipelineTemplateStages(r.Context(), template.Id)
	if err != nil {
		s.logger.Error("load pipeline template stages failed", "template_id", template.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template stages"})
		return
	}
	duplicated := repository.PipelineTemplate{Id: repository.NewId(), ProjectId: template.ProjectId, Name: name, Description: template.Description, VariableDeclarations: template.VariableDeclarations, Version: 1}
	stages := make([]repository.PipelineTemplateStage, 0, len(existingStages))
	for _, stage := range existingStages {
		stage.Id = repository.NewId()
		stage.TemplateId = duplicated.Id
		stages = append(stages, stage)
	}
	if err := s.store.DuplicatePipelineTemplate(r.Context(), duplicated, stages); err != nil {
		s.logger.Error("duplicate pipeline template failed", "template_id", template.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to duplicate pipeline template"})
		return
	}
	created, err := s.store.PipelineTemplate(r.Context(), duplicated.Id)
	if err != nil {
		s.logger.Error("load duplicated pipeline template failed", "template_id", duplicated.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template"})
		return
	}
	s.writePipelineTemplateDetail(w, r, created, http.StatusCreated)
}

func (s Server) resolvePipelineTemplateVariables(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	var req templateVariableResolveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	stages, ok := s.loadBuildStagesForOrchestration(w, r, projectId, req.Orchestration)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, resolveTemplateVariables(stages, sanitizePipelineTemplateVariables(req.VariableDeclarations)))
}

func (s Server) loadPipelineTemplateForCurrentUser(w http.ResponseWriter, r *http.Request) (repository.PipelineTemplate, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return repository.PipelineTemplate{}, false
	}
	templateId := urlParam(r, "template_id")
	template, err := s.store.PipelineTemplate(r.Context(), templateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Pipeline template " + templateId + " not found"})
			return repository.PipelineTemplate{}, false
		}
		s.logger.Error("load pipeline template failed", "template_id", templateId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template"})
		return repository.PipelineTemplate{}, false
	}
	if !s.ensureProjectMembership(w, r, pipelineTemplateProjectId(template), current.Id) {
		return repository.PipelineTemplate{}, false
	}
	return template, true
}

func (s Server) writePipelineTemplateDetail(w http.ResponseWriter, r *http.Request, template repository.PipelineTemplate, status int) {
	resp, ok := s.pipelineTemplateResponse(w, r, template)
	if !ok {
		return
	}
	writeJSON(w, status, resp)
}

func (s Server) pipelineTemplateResponse(w http.ResponseWriter, r *http.Request, template repository.PipelineTemplate) (pipelineTemplateResp, bool) {
	orchestration, ok := s.pipelineTemplateOrchestration(w, r, template.Id)
	if !ok {
		return pipelineTemplateResp{}, false
	}
	stages, ok := s.pipelineTemplateStagesResponse(w, r, pipelineTemplateProjectId(template), orchestration)
	if !ok {
		return pipelineTemplateResp{}, false
	}
	variables, ok := pipelineTemplateVariables(w, template.VariableDeclarations)
	if !ok {
		return pipelineTemplateResp{}, false
	}
	return pipelineTemplateResp{Id: template.Id, Name: template.Name, Description: template.Description, Orchestration: orchestration, Stages: stages, VariableDeclarations: resolveTemplateVariablesFromResponses(stages, variables), Version: template.Version, CreatedAt: formatTime(template.CreatedAt), UpdatedAt: formatTime(template.UpdatedAt)}, true
}

func (s Server) pipelineTemplateOrchestration(w http.ResponseWriter, r *http.Request, templateId string) ([]stageOrchestrationResp, bool) {
	rows, err := s.store.PipelineTemplateStages(r.Context(), templateId)
	if err != nil {
		s.logger.Error("load pipeline template stages failed", "template_id", templateId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template stages"})
		return nil, false
	}
	orchestration := make([]stageOrchestrationResp, 0, len(rows))
	for _, row := range rows {
		dependsOn, ok := pipelineTemplateDependsOn(w, row.DependsOn)
		if !ok {
			return nil, false
		}
		orchestration = append(orchestration, stageOrchestrationResp{StageId: row.StageId, StageName: row.StageName, StageVersion: row.StageVersion, DependsOn: dependsOn, SortOrder: row.SortOrder})
	}
	return orchestration, true
}

func (s Server) pipelineTemplateStagesResponse(w http.ResponseWriter, r *http.Request, projectId string, orchestration []stageOrchestrationResp) ([]buildStageResp, bool) {
	stageIds := make([]string, 0, len(orchestration))
	for _, item := range orchestration {
		stageIds = append(stageIds, item.StageId)
	}
	stages, err := s.store.BuildStagesByIds(r.Context(), projectId, stageIds)
	if err != nil {
		s.logger.Error("load pipeline template build stages failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load build stages"})
		return nil, false
	}
	stageMap := make(map[string]repository.BuildStage, len(stages))
	for _, stage := range stages {
		stageMap[stage.Id] = stage
	}
	responses := make([]buildStageResp, 0, len(orchestration))
	for _, item := range orchestration {
		stage, exists := stageMap[item.StageId]
		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Stage " + item.StageId + " not found"})
			return nil, false
		}
		responses = append(responses, buildStageResponse(stage))
	}
	return responses, true
}

func (s Server) ensurePipelineTemplateNameAvailable(w http.ResponseWriter, r *http.Request, projectId string, name string, currentTemplateId string) bool {
	existing, err := s.store.PipelineTemplateByName(r.Context(), projectId, name)
	if err == nil {
		if existing.Id != currentTemplateId {
			writeJSON(w, http.StatusConflict, map[string]string{"detail": "Template '" + name + "' already exists"})
			return false
		}
		return true
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check pipeline template name failed", "template_name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check pipeline template name"})
		return false
	}
	return true
}

func (s Server) nextPipelineTemplateCopyName(w http.ResponseWriter, r *http.Request, projectId string, name string) (string, bool) {
	baseName := pipelineTemplateCopyPattern.ReplaceAllString(name, "")
	for i := 1; ; i++ {
		candidate := baseName + " copy"
		if i > 1 {
			candidate = fmt.Sprintf("%s copy %d", baseName, i)
		}
		_, err := s.store.PipelineTemplateByName(r.Context(), projectId, candidate)
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, true
		}
		if err != nil {
			s.logger.Error("check duplicate pipeline template name failed", "template_name", candidate, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check pipeline template name"})
			return "", false
		}
	}
}

func (s Server) applyPipelineTemplateUpdateReq(w http.ResponseWriter, r *http.Request, template *repository.PipelineTemplate, req map[string]json.RawMessage) ([]repository.PipelineTemplateStage, bool, bool) {
	projectId := pipelineTemplateProjectId(*template)
	versionChanged := false
	orchestrationProvided := false
	stages := []repository.PipelineTemplateStage(nil)
	if raw, exists := req["name"]; exists {
		value, ok := decodeRequiredString(w, raw, "Invalid pipeline template fields")
		if !ok || !s.ensurePipelineTemplateNameAvailable(w, r, projectId, value, template.Id) {
			return nil, false, false
		}
		if value != template.Name {
			template.Name = value
			versionChanged = true
		}
	}
	if raw, exists := req["description"]; exists {
		var value string
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &value); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid pipeline template fields"})
				return nil, false, false
			}
		}
		if value != template.Description {
			template.Description = value
			versionChanged = true
		}
	}
	if raw, exists := req["variable_declarations"]; exists && string(raw) != "null" {
		var variables []map[string]any
		if err := json.Unmarshal(raw, &variables); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid pipeline template fields"})
			return nil, false, false
		}
		value, ok := marshalPipelineTemplateVariables(w, sanitizePipelineTemplateVariables(variables))
		if !ok {
			return nil, false, false
		}
		if value != template.VariableDeclarations {
			template.VariableDeclarations = value
			versionChanged = true
		}
	}
	if raw, exists := req["orchestration"]; exists && string(raw) != "null" {
		orchestrationProvided = true
		var orchestration []stageOrchestrationResp
		if err := json.Unmarshal(raw, &orchestration); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid pipeline template fields"})
			return nil, false, false
		}
		loaded, ok := s.loadBuildStagesForOrchestration(w, r, projectId, orchestration)
		if !ok {
			return nil, false, false
		}
		stages, ok = pipelineTemplateStageRows(w, template.Id, orchestration, loaded)
		if !ok {
			return nil, false, false
		}
		versionChanged = true
	}
	if versionChanged {
		template.Version++
	}
	return stages, orchestrationProvided, true
}

func (s Server) loadBuildStagesForOrchestration(w http.ResponseWriter, r *http.Request, projectId string, orchestration []stageOrchestrationResp) ([]repository.BuildStage, bool) {
	stageIds := make([]string, 0, len(orchestration))
	seen := map[string]struct{}{}
	for _, item := range orchestration {
		stageId := strings.TrimSpace(item.StageId)
		if stageId == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid pipeline template fields"})
			return nil, false
		}
		if _, exists := seen[stageId]; !exists {
			stageIds = append(stageIds, stageId)
			seen[stageId] = struct{}{}
		}
	}
	stages, err := s.store.BuildStagesByIds(r.Context(), projectId, stageIds)
	if err != nil {
		s.logger.Error("load build stages failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load build stages"})
		return nil, false
	}
	if len(stages) != len(stageIds) {
		found := map[string]struct{}{}
		for _, stage := range stages {
			found[stage.Id] = struct{}{}
		}
		missing := make([]string, 0)
		for _, stageId := range stageIds {
			if _, exists := found[stageId]; !exists {
				missing = append(missing, stageId)
			}
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Stage(s) not found: " + strings.Join(missing, ", ")})
		return nil, false
	}
	return stages, true
}

func pipelineTemplateStageRows(w http.ResponseWriter, templateId string, orchestration []stageOrchestrationResp, stages []repository.BuildStage) ([]repository.PipelineTemplateStage, bool) {
	stageMap := make(map[string]repository.BuildStage, len(stages))
	for _, stage := range stages {
		stageMap[stage.Id] = stage
	}
	rows := make([]repository.PipelineTemplateStage, 0, len(orchestration))
	for _, item := range orchestration {
		stageId := strings.TrimSpace(item.StageId)
		dependsOn, ok := marshalPipelineTemplateDependsOn(w, item.DependsOn)
		if !ok {
			return nil, false
		}
		stage := stageMap[stageId]
		rows = append(rows, repository.PipelineTemplateStage{Id: repository.NewId(), TemplateId: templateId, StageId: stage.Id, StageName: stage.Name, StageVersion: stage.Version, DependsOn: dependsOn, SortOrder: item.SortOrder})
	}
	return rows, true
}

func normalizePipelineTemplateCreateReq(w http.ResponseWriter, req *pipelineTemplateCreateReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid pipeline template fields"})
		return false
	}
	return true
}

func pipelineTemplateProjectId(template repository.PipelineTemplate) string {
	if template.ProjectId == nil {
		return ""
	}
	return *template.ProjectId
}

func pipelineTemplateVariables(w http.ResponseWriter, value string) ([]map[string]any, bool) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, true
	}
	var variables []map[string]any
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Invalid pipeline template variables"})
		return nil, false
	}
	return variables, true
}

func marshalPipelineTemplateVariables(w http.ResponseWriter, variables []map[string]any) (string, bool) {
	if variables == nil {
		variables = []map[string]any{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid variable_declarations"})
		return "", false
	}
	return string(data), true
}

func pipelineTemplateDependsOn(w http.ResponseWriter, value string) ([]string, bool) {
	if strings.TrimSpace(value) == "" {
		return []string{}, true
	}
	var dependsOn []string
	if err := json.Unmarshal([]byte(value), &dependsOn); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Invalid pipeline template orchestration"})
		return nil, false
	}
	return dependsOn, true
}

func marshalPipelineTemplateDependsOn(w http.ResponseWriter, dependsOn []string) (string, bool) {
	if dependsOn == nil {
		dependsOn = []string{}
	}
	data, err := json.Marshal(dependsOn)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid pipeline template fields"})
		return "", false
	}
	return string(data), true
}

func sanitizePipelineTemplateVariables(variables []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(variables))
	for _, variable := range variables {
		name, _ := variable["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || isPipelineTemplateBuiltinVariable(name) {
			continue
		}
		source, _ := variable["source"].(string)
		if source == "" {
			source = "template_custom"
		}
		_, hasValue := variable["value"]
		if source != "template_custom" && source != "repository_custom" && !(source == "template_stage" && hasValue && variable["value"] != nil) {
			continue
		}
		copy := map[string]any{}
		for key, value := range variable {
			copy[key] = value
		}
		copy["name"] = name
		if source == "template_stage" {
			copy["source"] = "template_custom"
		} else {
			copy["source"] = source
		}
		copy["editable"] = true
		if _, exists := copy["secret"]; !exists {
			copy["secret"] = false
		}
		result = append(result, copy)
	}
	return result
}

func resolveTemplateVariablesFromResponses(stages []buildStageResp, custom []map[string]any) []map[string]any {
	converted := make([]repository.BuildStage, 0, len(stages))
	for _, stage := range stages {
		artifacts, _ := json.Marshal(stage.Artifacts)
		artifactString := string(artifacts)
		converted = append(converted, repository.BuildStage{Name: stage.Name, Script: stage.Script, Artifacts: &artifactString})
	}
	return resolveTemplateVariables(converted, custom)
}

func resolveTemplateVariables(stages []repository.BuildStage, custom []map[string]any) []map[string]any {
	extracted := map[string]any{}
	for _, stage := range stages {
		extractTemplateVariables(stage.Script, extracted)
		if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
			var artifacts []artifactConfigResp
			if err := json.Unmarshal([]byte(*stage.Artifacts), &artifacts); err == nil {
				for _, artifact := range artifacts {
					extractTemplateVariables(artifact.Path, extracted)
					extractTemplateVariables(artifact.Name, extracted)
				}
			}
		}
	}
	custom = sanitizePipelineTemplateVariables(custom)
	customByName := make(map[string]map[string]any, len(custom))
	for _, variable := range custom {
		name, _ := variable["name"].(string)
		customByName[name] = variable
	}
	builtinNames := sortedPipelineTemplateBuiltinVariableNames()
	result := []map[string]any{}
	if len(extracted) == 0 {
		for _, name := range builtinNames {
			result = append(result, pipelineTemplateBuiltinVariable(name))
		}
		for _, variable := range custom {
			name, _ := variable["name"].(string)
			if !isPipelineTemplateBuiltinVariable(name) {
				result = append(result, variable)
			}
		}
		return result
	}
	extractedNames := make([]string, 0, len(extracted))
	for name := range extracted {
		extractedNames = append(extractedNames, name)
	}
	sort.Strings(extractedNames)
	for _, name := range extractedNames {
		if isPipelineTemplateBuiltinVariable(name) {
			result = append(result, pipelineTemplateBuiltinVariable(name))
			continue
		}
		if existing, ok := customByName[name]; ok {
			if _, exists := existing["default"]; !exists && extracted[name] != nil {
				existing["default"] = extracted[name]
			}
			result = append(result, existing)
			continue
		}
		result = append(result, map[string]any{"name": name, "description": "", "default": extracted[name], "value": nil, "secret": false, "source": "template_stage", "editable": true})
	}
	return result
}

func extractTemplateVariables(text string, found map[string]any) {
	for _, match := range templateVariablePattern.FindAllStringSubmatch(text, -1) {
		name := match[1]
		defaultValue := any(nil)
		if len(match) > 2 && match[2] != "" {
			defaultValue = match[2]
		}
		if current, exists := found[name]; !exists || current == nil && defaultValue != nil {
			found[name] = defaultValue
		}
	}
}

func isPipelineTemplateBuiltinVariable(name string) bool {
	_, ok := pipelineTemplateBuiltinVariableSpecs()[name]
	return ok
}

func sortedPipelineTemplateBuiltinVariableNames() []string {
	names := make([]string, 0, len(pipelineTemplateBuiltinVariableSpecs()))
	for name := range pipelineTemplateBuiltinVariableSpecs() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func pipelineTemplateBuiltinVariable(name string) map[string]any {
	spec := pipelineTemplateBuiltinVariableSpecs()[name]
	return map[string]any{"name": name, "description": spec, "default": nil, "value": nil, "secret": false, "source": "template", "editable": false}
}

func pipelineTemplateBuiltinVariableSpecs() map[string]string {
	return map[string]string{
		"repository_id":    "运行时注入: 当前项目 ID",
		"repository_name":  "运行时注入: 当前项目名称",
		"repository_code":  "运行时注入: 当前项目编码",
		"repository_url":   "运行时注入: 当前仓库地址",
		"repository_ref":   "运行时注入: 当前分支",
		"template_id":      "运行时注入: 当前模板 ID",
		"template_name":    "运行时注入: 当前模板名称",
		"template_version": "运行时注入: 当前模板版本",
		"runtime_datetime": "运行时注入: 流水线启动时间 (UTC, 格式 YYYYmmdd-HHmmss)",
	}
}
