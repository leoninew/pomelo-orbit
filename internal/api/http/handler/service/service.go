package servicehandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"github.com/gin-gonic/gin"
)

func (h Handler) ListServices(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListServices(c.Request.Context(), current.Id, servicedto.ServiceListInput{
		ProjectId:     c.Request.URL.Query().Get("project_id"),
		ApplicationId: c.Request.URL.Query().Get("application_id"),
		EnvironmentId: c.Request.URL.Query().Get("environment_id"),
		Status:        c.Request.URL.Query().Get("status"),
		Search:        c.Request.URL.Query().Get("search"),
		Page:          page,
		PerPage:       perPage,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := serviceViewResponses(items.Items)
	transportresponse.ProtoJSON(c, http.StatusOK, &servicev1.ServicePaginatedResp{
		Items:   transportresponse.Ptrs(resp),
		Total:   int32(items.Total),
		Page:    int32(items.Page),
		PerPage: int32(items.PerPage),
		Pages:   int32(transportresponse.PageCount(items.Total, items.PerPage)),
	})
}

func (h Handler) GetService(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.GetService(c.Request.Context(), current.Id, c.Param("service_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := serviceViewResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

// ListApplicationServices uses the service domain's authorization and query path.
func (h Handler) ListApplicationServices(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListServicesByApplication(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationServiceResponses(items)
	transportresponse.ProtoJSON(c, http.StatusOK, &servicev1.ServiceListResp{Items: transportresponse.Ptrs(resp)})
}

func applicationServiceResponses(items []model.Service) []servicev1.ServiceResp {
	resp := make([]servicev1.ServiceResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, servicev1.ServiceResp{
			Id: item.Id, ApplicationId: item.ApplicationId, EnvironmentId: item.EnvironmentId,
			InstanceKey: item.InstanceKey, VersionId: item.VersionId,
			LastSuccessfulVersionId: item.LastSuccessfulVersionId, Status: item.Status,
			CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt),
		})
	}
	return resp
}
