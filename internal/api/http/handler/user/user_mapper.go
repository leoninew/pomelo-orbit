package userhandler

import (
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	userdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func userCreateInput(req *pomeloorbit.UserCreateReq) userdto.CreateInput {
	return userdto.CreateInput{Username: req.Username, Password: req.Password, Email: req.Email}
}

func userUpdateInput(req *pomeloorbit.UserUpdateReq) userdto.UpdateInput {
	return userdto.UpdateInput{Username: req.Username, Password: req.Password, Status: req.Status}
}

func actor(current authz.CurrentUserContext) userdto.Actor {
	return userdto.Actor{UserId: current.User.Id, Permissions: current.Permissions}
}

func userDetailResponse(detail userdto.Detail) pomeloorbit.UserResp {
	user := detail.User
	return pomeloorbit.UserResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roleCodes(detail.Roles), Permissions: emptyStrings(detail.Permissions), Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(detail.Roles, detail.PermissionsByRoleId))}
}

func userListResponse(item userdto.ListItem) pomeloorbit.UserListResp {
	user := item.User
	return pomeloorbit.UserListResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(item.Roles, nil))}
}

func roleCodes(roles []model.Role) []string {
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	return codes
}

func roleResponses(roles []model.Role, permissionsByRoleID map[string][]string) []pomeloorbit.UserRoleResp {
	items := make([]pomeloorbit.UserRoleResp, 0, len(roles))
	for _, role := range roles {
		permissionCodes := []string{}
		if permissionsByRoleID != nil && permissionsByRoleID[role.Id] != nil {
			permissionCodes = permissionsByRoleID[role.Id]
		}
		items = append(items, pomeloorbit.UserRoleResp{Id: role.Id, Code: role.Code, Name: role.Name, PermissionCodes: permissionCodes})
	}
	return items
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
