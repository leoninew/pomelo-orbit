package applicationhandler

import (
	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
)

func applicationImportInput(projectId string, req *applicationv1.ApplicationImportReq) applicationdto.ApplicationImportInput {
	components := make([]applicationdto.VersionComponentInput, 0, len(req.Components))
	for _, item := range req.Components {
		components = append(components, versionComponentInput(item))
	}
	kind := ""
	if req.Kind != nil {
		kind = *req.Kind
	}
	return applicationdto.ApplicationImportInput{
		ProjectId:    projectId,
		Name:         req.Name,
		Code:         req.Code,
		Kind:         kind,
		VersionLabel: req.VersionLabel,
		VersionNote:  req.VersionNote,
		Components:   components,
	}
}

func applicationExportResponse(exported applicationdto.ApplicationExport) *applicationv1.ApplicationExportResp {
	resp := &applicationv1.ApplicationExportResp{
		Id:        exported.Application.Id,
		ProjectId: exported.Application.ProjectId,
		Name:      exported.Application.Name,
		Code:      exported.Application.Code,
		Kind:      exported.Application.Kind,
		Versions:  make([]*applicationv1.VersionResp, 0, len(exported.Versions)),
		Services:  make([]*servicev1.ServiceResp, 0, len(exported.Services)),
	}
	for _, view := range exported.Versions {
		item := versionResponse(view)
		resp.Versions = append(resp.Versions, &item)
	}
	for _, svc := range exported.Services {
		item := serviceResponse(svc)
		resp.Services = append(resp.Services, &item)
	}
	return resp
}
