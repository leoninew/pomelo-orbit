package applicationhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	applicationv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/application"
)

func (h Handler) GetVersionComponent(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	component, err := h.service.VersionComponentForUser(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) CreateVersionComponent(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.CreateVersionComponent(c.Request.Context(), current.Id, c.Param("version_id"), versionComponentCreateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusCreated, resp)
}

func (h Handler) UpdateVersionComponentBasic(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentBasicUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentBasic(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentBasicUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) UpdateVersionComponentRuntime(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentRuntimeUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentRuntime(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentRuntimeUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) UpdateVersionComponentEndpoints(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentEndpointsUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentEndpoints(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentEndpointsUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) UpdateVersionComponentEnv(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentEnvUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentEnv(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentEnvUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) UpdateVersionComponentMounts(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentMountsUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentMounts(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentMountsUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) UpdateVersionComponentDependencies(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentDependenciesUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentDependencies(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentDependenciesUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) UpdateVersionComponentDevices(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentDevicesUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentDevices(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentDevicesUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) UpdateVersionComponentAdvanced(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionComponentAdvancedUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	component, err := h.service.UpdateVersionComponentAdvanced(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id"), versionComponentAdvancedUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionComponentResponse(component)
	transportresponse.ProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) DeleteVersionComponent(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteVersionComponent(c.Request.Context(), current.Id, c.Param("version_id"), c.Param("component_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
