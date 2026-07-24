package projecthandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	projectdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/project/dto"
	projectv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/project"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func projectSaveInput(req *projectv1.ProjectSaveReq) projectdto.SaveInput {
	return projectdto.SaveInput{Name: req.Name, Code: req.Code}
}

func projectResponse(project model.Project) projectv1.ProjectResp {
	return projectv1.ProjectResp{Id: project.Id, Name: project.Name, Code: project.Code, IsActive: project.IsActive, CreatedAt: transportresponse.FormatTime(project.CreatedAt), UpdatedAt: transportresponse.FormatTime(project.UpdatedAt)}
}

func projectMemberResponses(users []model.User) []projectv1.ProjectMemberResp {
	resp := make([]projectv1.ProjectMemberResp, 0, len(users))
	for _, user := range users {
		resp = append(resp, projectMemberResponse(user))
	}
	return resp
}

func projectMemberResponse(user model.User) projectv1.ProjectMemberResp {
	return projectv1.ProjectMemberResp{Id: user.Id, Username: user.Username, Email: user.Email, Status: user.Status, AuthSource: user.AuthSource, LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt)}
}
