package rolehandler

import transportresponse "backend/internal/transport/http/response"

type PermissionResp struct {
	Id          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type PermissionListResp = transportresponse.ListResp[PermissionResp]

type RoleResp struct {
	Id              string   `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	PermissionCodes []string `json:"permission_codes"`
}

type RoleCreateReq struct {
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
}

type RoleUpdateReq struct {
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Description     *string  `json:"description"`
	PermissionCodes []string `json:"permission_codes"`
}
