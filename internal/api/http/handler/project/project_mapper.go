package projecthandler

import (
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	projectv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/project"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func projectCreateInput(req *projectv1.ProjectCreateReq) projectdto.CreateInput {
	input := projectdto.CreateInput{Name: req.Name, Code: req.Code}
	if req.Environment == nil {
		return input
	}
	input.Environment = projectdto.EnvironmentCreateInput{
		State:                      req.Environment.State,
		Platform:                   req.Environment.Platform,
		Host:                       req.Environment.Host,
		Port:                       int(req.Environment.Port),
		Username:                   req.Environment.Username,
		WorkspaceRoot:              req.Environment.WorkspaceRoot,
		DeploymentSSHKeyName:       req.Environment.DeploymentSshKeyName,
		DeploymentSSHPrivateKey:    req.Environment.DeploymentSshPrivateKey,
		DeploymentSSHKeyPassphrase: optionalString(req.Environment.DeploymentSshKeyPassphrase),
		HostKeyFingerprint:         req.Environment.HostKeyFingerprint,
	}
	return input
}

func projectSaveInput(req *projectv1.ProjectSaveReq) projectdto.SaveInput {
	return projectdto.SaveInput{Name: req.Name}
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
