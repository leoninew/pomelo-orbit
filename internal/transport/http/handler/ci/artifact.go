package cihandler

import (
	pomeloorbit "gitee.com/leoninew/pomelo-orbit/internal/gen/proto/orbit"
	"net/http"

	cisvc "gitee.com/leoninew/pomelo-orbit/internal/service/ci"
	transportresponse "gitee.com/leoninew/pomelo-orbit/internal/transport/http/response"
)

func (h Handler) RegisterArtifactRoutes(r router) {
	r.Get("/api/ci/artifact", h.listArtifacts)
}

func (h Handler) listArtifacts(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListArtifacts(r.Context(), current.Id, cisvc.ArtifactListInput{ProjectId: r.URL.Query().Get("project_id"), RepositoryId: r.URL.Query().Get("repository_id"), TemplateId: r.URL.Query().Get("template_id"), Search: r.URL.Query().Get("search"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := mapPage(items, artifactResponse)
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.ArtifactPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}
