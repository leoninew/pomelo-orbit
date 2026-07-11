package settingshandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	settingsdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
)

func configUpdateValue(req *pomeloorbit.SystemConfigUpdateReq) any {
	return transportresponse.NativeValue(req.Value)
}

func systemConfigResponse(config settingsdto.SystemConfig) pomeloorbit.SystemConfigResp {
	items := make([]pomeloorbit.ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return pomeloorbit.SystemConfigResp{Items: transportresponse.Ptrs(items)}
}

func configItemResponse(item settingsdto.ConfigItem) pomeloorbit.ConfigItemResp {
	return pomeloorbit.ConfigItemResp{Key: item.Key, Value: transportresponse.ProtoValue(item.Value), Default: transportresponse.ProtoValue(item.Default), IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description}
}
