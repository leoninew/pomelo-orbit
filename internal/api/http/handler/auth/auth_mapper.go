package authhandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	authdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func loginInput(req *pomeloorbit.LoginReq, ip string, userAgent string) authdto.LoginInput {
	return authdto.LoginInput{Username: req.Username, Password: req.Password, CSRFToken: req.CsrfToken, IP: ip, UserAgent: userAgent}
}

func changePasswordInput(user model.User, req *pomeloorbit.PasswordChangeReq) authdto.ChangePasswordInput {
	return authdto.ChangePasswordInput{User: user, OldPassword: req.OldPassword, NewPassword: req.NewPassword}
}

func csrfTokenResponse(token string) pomeloorbit.CSRFTokenResp {
	return pomeloorbit.CSRFTokenResp{Token: token}
}

func turnstileConfigResponse(enabled bool, siteKey string) pomeloorbit.TurnstileConfigResp {
	return pomeloorbit.TurnstileConfigResp{Enabled: enabled, SiteKey: siteKey}
}

func tokenResponse(token string) pomeloorbit.TokenResp {
	return pomeloorbit.TokenResp{AccessToken: token, TokenType: "bearer"}
}

func userInfoResponse(user model.User, roles []string, permissions []string) pomeloorbit.UserInfoResp {
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	return pomeloorbit.UserInfoResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roles, Permissions: permissions}
}

func loginHistoryResponse(history model.LoginHistory) pomeloorbit.LoginHistoryResp {
	return pomeloorbit.LoginHistoryResp{Id: history.Id, UserId: history.UserId, Username: history.Username, IpAddress: history.IpAddress, UserAgent: history.UserAgent, LoginAt: transportresponse.FormatTime(history.LoginAt), Success: history.Success}
}
