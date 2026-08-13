package rolehandler

import (
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	roledto "github.com/leoninew/pomelo-orbit/internal/application/role/dto"
	rolev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/role"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func roleCreateInput(req *rolev1.RoleCreateReq) roledto.SaveInput {
	return roledto.SaveInput{Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes}
}

func roleUpdateInput(req *rolev1.RoleUpdateReq) roledto.SaveInput {
	return roledto.SaveInput{Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes}
}

func roleDetailResponse(detail roledto.Detail) rolev1.RoleResp {
	return roleResponse(detail.Role, detail.PermissionCodes)
}

func roleResponse(role model.Role, permissionCodes []string) rolev1.RoleResp {
	if permissionCodes == nil {
		permissionCodes = []string{}
	}
	return rolev1.RoleResp{Id: role.Id, Code: role.Code, Name: role.Name, Description: role.Description, CreatedAt: transportresponse.FormatTime(role.CreatedAt), UpdatedAt: transportresponse.FormatTime(role.UpdatedAt), PermissionCodes: permissionCodes}
}

func permissionResponse(permission roledto.Permission) rolev1.PermissionResp {
	return rolev1.PermissionResp{Id: permission.Id, Code: permission.Code, Name: permission.Name, Description: permission.Description}
}
