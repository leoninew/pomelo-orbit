package userhandler

type UserRoleResp struct {
	Id              string   `json:"id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	PermissionCodes []string `json:"permission_codes"`
}

type UserListResp struct {
	Id          string         `json:"id"`
	Username    string         `json:"username"`
	Email       *string        `json:"email"`
	AuthSource  string         `json:"auth_source"`
	CreatedAt   string         `json:"created_at"`
	LastLoginAt *string        `json:"last_login_at"`
	Status      string         `json:"status"`
	UpdatedAt   string         `json:"updated_at"`
	RoleItems   []UserRoleResp `json:"role_items"`
}

type UserResp struct {
	Id          string         `json:"id"`
	Username    string         `json:"username"`
	Email       *string        `json:"email"`
	AuthSource  string         `json:"auth_source"`
	CreatedAt   string         `json:"created_at"`
	LastLoginAt *string        `json:"last_login_at"`
	Roles       []string       `json:"roles"`
	Permissions []string       `json:"permissions"`
	Status      string         `json:"status"`
	UpdatedAt   string         `json:"updated_at"`
	RoleItems   []UserRoleResp `json:"role_items"`
}

type UserCreateReq struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Email    *string `json:"email"`
}

type UserUpdateReq struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	Status   *string `json:"status"`
}

type UserRoleUpdateReq struct {
	RoleIds []string `json:"role_ids"`
}
