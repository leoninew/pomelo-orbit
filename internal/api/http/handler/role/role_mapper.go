package rolehandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	roledto "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func roleCreateInput(req *pomeloorbit.RoleCreateReq) roledto.SaveInput {
	return roledto.SaveInput{Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes}
}

func roleUpdateInput(req *pomeloorbit.RoleUpdateReq) roledto.SaveInput {
	return roledto.SaveInput{Code: req.Code, Name: req.Name, Description: req.Description, PermissionCodes: req.PermissionCodes}
}

func roleDetailResponse(detail roledto.Detail) pomeloorbit.RoleResp {
	return roleResponse(detail.Role, detail.PermissionCodes)
}

func roleResponse(role model.Role, permissionCodes []string) pomeloorbit.RoleResp {
	if permissionCodes == nil {
		permissionCodes = []string{}
	}
	return pomeloorbit.RoleResp{Id: role.Id, Code: role.Code, Name: role.Name, Description: role.Description, CreatedAt: transportresponse.FormatTime(role.CreatedAt), UpdatedAt: transportresponse.FormatTime(role.UpdatedAt), PermissionCodes: permissionCodes}
}

func permissionResponse(permission roledto.Permission) pomeloorbit.PermissionResp {
	return pomeloorbit.PermissionResp{Id: permission.Id, Code: permission.Code, Name: permission.Name, Description: permission.Description}
}
