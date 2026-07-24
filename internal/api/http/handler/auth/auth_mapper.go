package authhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	authdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/dto"
	authv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/auth"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func loginInput(req *authv1.LoginReq, ip string, userAgent string) authdto.LoginInput {
	return authdto.LoginInput{Username: req.Username, Password: req.Password, CSRFToken: req.CsrfToken, IP: ip, UserAgent: userAgent}
}

func changePasswordInput(user model.User, req *authv1.PasswordChangeReq) authdto.ChangePasswordInput {
	return authdto.ChangePasswordInput{User: user, OldPassword: req.OldPassword, NewPassword: req.NewPassword}
}

func csrfTokenResponse(token string) authv1.CSRFTokenResp {
	return authv1.CSRFTokenResp{Token: token}
}

func turnstileConfigResponse(enabled bool, siteKey string) authv1.TurnstileConfigResp {
	return authv1.TurnstileConfigResp{Enabled: enabled, SiteKey: siteKey}
}

func tokenResponse(token string) authv1.TokenResp {
	return authv1.TokenResp{AccessToken: token, TokenType: "bearer"}
}

func userInfoResponse(user model.User, roles []string, permissions []string) authv1.UserInfoResp {
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	return authv1.UserInfoResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roles, Permissions: permissions}
}

func loginHistoryResponse(history model.LoginHistory) authv1.LoginHistoryResp {
	return authv1.LoginHistoryResp{Id: history.Id, UserId: history.UserId, Username: history.Username, IpAddress: history.IpAddress, UserAgent: history.UserAgent, LoginAt: transportresponse.FormatTime(history.LoginAt), Success: history.Success}
}
