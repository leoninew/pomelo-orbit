package cdhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func applicationImportInput(projectID string, req *pomeloorbit.ApplicationImportReq) cddto.ApplicationImportInput {
	files := make([]cddto.ConfigFileInput, 0, len(req.ConfigFiles))
	for _, file := range req.ConfigFiles {
		files = append(files, configFileInput(file))
	}
	serviceConfigs := make([]cddto.ApplicationServiceConfigImportInput, 0, len(req.ServiceConfigs))
	for _, item := range req.ServiceConfigs {
		serviceConfigs = append(serviceConfigs, cddto.ApplicationServiceConfigImportInput{
			ServiceName: item.ServiceName,
			Image:       item.Image,
			Environment: item.Environment,
			Volumes:     item.Volumes,
		})
	}
	routes := make([]cddto.ApplicationRouteInput, 0, len(req.Routes))
	for _, item := range req.Routes {
		routes = append(routes, applicationRouteInput(item))
	}
	return cddto.ApplicationImportInput{
		ProjectId:         projectID,
		Version:           req.Version,
		Name:              req.Name,
		Code:              req.Code,
		ImagePullPolicy:   req.ImagePullPolicy,
		RouteManaged:      req.RouteManaged,
		ConfigFiles:       files,
		ServiceConfigs:    serviceConfigs,
		ApplicationRoutes: routes,
	}
}

func applicationExportResponse(exported cddto.ApplicationExport) *pomeloorbit.ApplicationExportResp {
	resp := &pomeloorbit.ApplicationExportResp{
		Version:         "1.0",
		Name:            exported.Application.Name,
		Code:            exported.Application.Code,
		ImagePullPolicy: exported.Application.ImagePullPolicy,
		RouteManaged:    exported.Application.RouteManaged,
		ConfigFiles:     make([]*pomeloorbit.ApplicationExportConfigFileResp, 0, len(exported.ConfigFiles)),
		ServiceConfigs:  make([]*pomeloorbit.ApplicationServiceConfigExportResp, 0, len(exported.ServiceConfigs)),
		Routes:          make([]*pomeloorbit.ApplicationExportRouteResp, 0, len(exported.Routes)),
	}
	for _, file := range exported.ConfigFiles {
		resp.ConfigFiles = append(resp.ConfigFiles, &pomeloorbit.ApplicationExportConfigFileResp{Path: file.Path, Content: file.Content})
	}
	for _, config := range exported.ServiceConfigs {
		resp.ServiceConfigs = append(resp.ServiceConfigs, &pomeloorbit.ApplicationServiceConfigExportResp{
			ServiceName: config.ServiceName,
			Image:       config.Image,
			Environment: config.Environment,
			Volumes:     config.Volumes,
		})
	}
	for _, route := range exported.Routes {
		resp.Routes = append(resp.Routes, &pomeloorbit.ApplicationExportRouteResp{ServiceName: route.ServiceName, Domain: route.Domain, Port: int32(route.Port)})
	}
	return resp
}

func configFileInput(req *pomeloorbit.ConfigFileReq) cddto.ConfigFileInput {
	return cddto.ConfigFileInput{Path: req.Path, Content: req.Content}
}

func configFileResponses(files []model.ApplicationConfigFile) []pomeloorbit.ConfigFileResp {
	resp := make([]pomeloorbit.ConfigFileResp, 0, len(files))
	for _, file := range files {
		resp = append(resp, configFileResponse(file))
	}
	return resp
}

func configFileResponse(file model.ApplicationConfigFile) pomeloorbit.ConfigFileResp {
	return pomeloorbit.ConfigFileResp{Id: file.Id, Path: file.Path, CreatedAt: transportresponse.FormatTime(file.CreatedAt)}
}

func applicationRouteInput(req *pomeloorbit.ApplicationRouteReq) cddto.ApplicationRouteInput {
	return cddto.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: int(req.Port)}
}

func applicationRouteResponses(routes []model.ApplicationRoute) []pomeloorbit.ApplicationRouteResp {
	resp := make([]pomeloorbit.ApplicationRouteResp, 0, len(routes))
	for _, route := range routes {
		resp = append(resp, applicationRouteResponse(route))
	}
	return resp
}

func applicationRouteResponse(route model.ApplicationRoute) pomeloorbit.ApplicationRouteResp {
	return pomeloorbit.ApplicationRouteResp{
		Id:          route.Id,
		ServiceName: route.ServiceName,
		Domain:      route.Domain,
		Port:        int32(route.Port),
		CreatedAt:   transportresponse.FormatTime(route.CreatedAt),
		UpdatedAt:   transportresponse.FormatTime(route.UpdatedAt),
	}
}

func composeServiceResponses(views []cddto.ApplicationServiceConfigView) []pomeloorbit.ComposeServiceResp {
	resp := make([]pomeloorbit.ComposeServiceResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, pomeloorbit.ComposeServiceResp{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: int32(view.DefaultPort)})
	}
	return resp
}

func applicationServiceConfigResponses(views []cddto.ApplicationServiceConfigView) []pomeloorbit.ApplicationServiceConfigResp {
	resp := make([]pomeloorbit.ApplicationServiceConfigResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, applicationServiceConfigResponse(view))
	}
	return resp
}

func applicationServiceConfigResponse(view cddto.ApplicationServiceConfigView) pomeloorbit.ApplicationServiceConfigResp {
	return pomeloorbit.ApplicationServiceConfigResp{
		ServiceName:   view.ServiceName,
		DefaultDomain: view.DefaultDomain,
		DefaultPort:   int32(view.DefaultPort),
		BaseImage:     view.BaseImage,
		Image:         view.Image,
		ConfigId:      view.ConfigId,
		CreatedAt:     view.CreatedAt,
		UpdatedAt:     view.UpdatedAt,
	}
}
