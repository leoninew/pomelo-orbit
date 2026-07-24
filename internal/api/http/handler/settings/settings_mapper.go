package settingshandler

import (
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	settingsdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/dto"
	settingsv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/settings"
)

func configUpdateValue(req *settingsv1.SystemConfigUpdateReq) any {
	return transportresponse.NativeValue(req.Value)
}

func systemConfigResponse(config settingsdto.SystemConfig) settingsv1.SystemConfigResp {
	items := make([]settingsv1.ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return settingsv1.SystemConfigResp{Items: transportresponse.Ptrs(items)}
}

func configItemResponse(item settingsdto.ConfigItem) settingsv1.ConfigItemResp {
	return settingsv1.ConfigItemResp{Key: item.Key, Value: transportresponse.ProtoValue(item.Value), Default: transportresponse.ProtoValue(item.Default), IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description}
}
