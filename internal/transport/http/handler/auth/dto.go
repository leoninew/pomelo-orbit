package authhandler

import apiv1 "backend/internal/transport/http/dto/proto/orbit/api/v1"

type LoginHistoryResp = apiv1.LoginHistoryResp
type LoginHistoryPaginatedResp = apiv1.LoginHistoryPaginatedResp
type TokenResp = apiv1.TokenResp
type UserInfoResp = apiv1.UserInfoResp
type CSRFTokenResp = apiv1.CSRFTokenResp
type LoginReq = apiv1.LoginReq
type PasswordChangeReq = apiv1.PasswordChangeReq
type GoogleCallbackReq = apiv1.GoogleCallbackReq
type TurnstileConfigResp = apiv1.TurnstileConfigResp
