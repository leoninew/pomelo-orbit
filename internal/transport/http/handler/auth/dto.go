package authhandler

type LoginHistoryResp struct {
	Id        string  `json:"id"`
	UserId    string  `json:"user_id"`
	Username  string  `json:"username"`
	IpAddress *string `json:"ip_address"`
	UserAgent *string `json:"user_agent"`
	LoginAt   string  `json:"login_at"`
	Success   bool    `json:"success"`
}

type TokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type UserInfoResp struct {
	Id          string   `json:"id"`
	Username    string   `json:"username"`
	Email       *string  `json:"email"`
	AuthSource  string   `json:"auth_source"`
	CreatedAt   string   `json:"created_at"`
	LastLoginAt *string  `json:"last_login_at"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type CSRFTokenResp struct {
	Token string `json:"token"`
}

type LoginReq struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	CSRFToken      string `json:"csrf_token"`
	TurnstileToken string `json:"turnstile_token"`
}

type PasswordChangeReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type GoogleCallbackReq struct {
	Code string `json:"code"`
}

type TurnstileConfigResp struct {
	Enabled bool   `json:"enabled"`
	SiteKey string `json:"site_key"`
}
