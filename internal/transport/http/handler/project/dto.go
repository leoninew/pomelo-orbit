package projecthandler

import transportresponse "backend/internal/transport/http/response"

type ProjectResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ProjectMemberResp struct {
	Id          string  `json:"id"`
	Username    string  `json:"username"`
	Email       *string `json:"email"`
	Status      string  `json:"status"`
	AuthSource  string  `json:"auth_source"`
	LastLoginAt *string `json:"last_login_at"`
}

type ProjectListResp = transportresponse.ListResp[ProjectResp]
type ProjectMemberListResp = transportresponse.ListResp[ProjectMemberResp]

type ProjectSaveReq struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type ProjectMemberReq struct {
	UserId string `json:"user_id"`
}
