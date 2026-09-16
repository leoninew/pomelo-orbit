package servicehandler

import (
	"net/http"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	servicev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/service"
)

func (h Handler) ListServices(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListServices(c.Request.Context(), current.Id, servicedto.ServiceListInput{
		ProjectId:     c.Request.URL.Query().Get("project_id"),
		ApplicationId: c.Request.URL.Query().Get("application_id"),
		Status:        c.Request.URL.Query().Get("status"),
		Search:        c.Request.URL.Query().Get("search"),
		Page:          page,
		PerPage:       perPage,
	})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceViewResponses(items.Items)
	transport.WriteProtoJSON(c, http.StatusOK, &servicev1.ServicePaginatedResp{
		Items:   transport.Ptrs(resp),
		Total:   int32(items.Total),
		Page:    int32(items.Page),
		PerPage: int32(items.PerPage),
		Pages:   int32(transport.PageCount(items.Total, items.PerPage)),
	})
}

func (h Handler) GetService(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.GetService(c.Request.Context(), current.Id, c.Param("service_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceViewResponse(view)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) CreateService(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServiceCreateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.CreateService(c.Request.Context(), current.Id, servicedto.ServiceCreateInput{
		ApplicationId: req.ApplicationId, VersionId: req.VersionId, InstanceKey: req.InstanceKey, Code: req.Code,
	})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceViewResponse(view)
	transport.WriteProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) DeleteService(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteService(c.Request.Context(), current.Id, c.Param("service_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) GetServiceComponent(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.GetServiceComponent(c.Request.Context(), current.Id, c.Param("service_id"), c.Param("component_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceComponentDetailResponse(detail)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateServiceComponentOverlay(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServiceComponentOverlayUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateServiceComponentOverlay(c.Request.Context(), current.Id, c.Param("service_id"), c.Param("component_id"), serviceComponentOverlayInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceComponentResponse(component)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateServiceBasic(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServiceBasicUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateServiceBasic(c.Request.Context(), current.Id, c.Param("service_id"), servicedto.ServiceBasicUpdateInput{
		VersionId: req.VersionId, InstanceKey: req.InstanceKey,
	})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceViewResponse(view)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateServiceEnv(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServiceEnvUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateServiceEnv(c.Request.Context(), current.Id, c.Param("service_id"), serviceEnvUpdateInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceViewResponse(view)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

// ListApplicationServices uses the service domain's authorization and query path.
func (h Handler) ListApplicationServices(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListServiceViewsByApplication(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := serviceViewResponses(items)
	transport.WriteProtoJSON(c, http.StatusOK, &servicev1.ServiceListResp{Items: transport.Ptrs(resp)})
}
