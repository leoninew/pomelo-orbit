package userhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	userdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/dto"
	userv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/user"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func userCreateInput(req *userv1.UserCreateReq) userdto.CreateInput {
	return userdto.CreateInput{Username: req.Username, Password: req.Password, Email: req.Email}
}

func userUpdateInput(req *userv1.UserUpdateReq) userdto.UpdateInput {
	return userdto.UpdateInput{Username: req.Username, Password: req.Password, Status: req.Status}
}

func actor(current security.CurrentUserContext) userdto.Actor {
	return userdto.Actor{UserId: current.User.Id, Permissions: current.Permissions}
}

func userDetailResponse(detail userdto.Detail) userv1.UserResp {
	user := detail.User
	return userv1.UserResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roleCodes(detail.Roles), Permissions: emptyStrings(detail.Permissions), Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(detail.Roles, detail.PermissionsByRoleId))}
}

func userListResponse(item userdto.ListItem) userv1.UserListResp {
	user := item.User
	return userv1.UserListResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(item.Roles, nil))}
}

func roleCodes(roles []model.Role) []string {
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	return codes
}

func roleResponses(roles []model.Role, permissionsByRoleId map[string][]string) []userv1.UserRoleResp {
	items := make([]userv1.UserRoleResp, 0, len(roles))
	for _, role := range roles {
		permissionCodes := []string{}
		if permissionsByRoleId != nil && permissionsByRoleId[role.Id] != nil {
			permissionCodes = permissionsByRoleId[role.Id]
		}
		items = append(items, userv1.UserRoleResp{Id: role.Id, Code: role.Code, Name: role.Name, PermissionCodes: permissionCodes})
	}
	return items
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
