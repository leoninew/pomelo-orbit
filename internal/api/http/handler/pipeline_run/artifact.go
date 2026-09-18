package pipelinerunhandler

import (
	"net/http"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	pipelinerunv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/pipeline_run"

	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
)

func (h Handler) ListArtifacts(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListArtifacts(c.Request.Context(), current.Id, pipelinerundto.ArtifactListInput{ProjectId: c.Request.URL.Query().Get("project_id"), RepositoryId: c.Request.URL.Query().Get("repository_id"), PipelineId: c.Request.URL.Query().Get("pipeline_id"), Search: c.Request.URL.Query().Get("search"), Page: page, PerPage: perPage})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := make([]pipelinerunv1.ArtifactResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, artifactResponse(item))
	}
	transport.WriteProtoJSON(c, http.StatusOK, &pipelinerunv1.ArtifactPaginatedResp{Items: transport.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transport.PageCount(items.Total, items.PerPage))})
}

func (h Handler) GetArtifact(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	item, err := h.service.ArtifactForUser(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("artifact_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := artifactResponse(item)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}
