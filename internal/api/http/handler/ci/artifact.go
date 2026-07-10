package cihandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
)

func (h Handler) ListArtifacts(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListArtifacts(c.Request.Context(), current.Id, cidto.ArtifactListInput{ProjectId: c.Request.URL.Query().Get("project_id"), RepositoryId: c.Request.URL.Query().Get("repository_id"), TemplateId: c.Request.URL.Query().Get("template_id"), Search: c.Request.URL.Query().Get("search"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.ArtifactResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, artifactResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ArtifactPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}
