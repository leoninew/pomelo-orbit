package settingshandler

import (
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	settingsdto "github.com/leoninew/pomelo-orbit/internal/application/settings/dto"
	settingsv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/settings"
)

func configUpdateValue(req *settingsv1.SystemConfigUpdateReq) any {
	return transport.NativeValue(req.Value)
}

func systemConfigResponse(config settingsdto.SystemConfig) settingsv1.SystemConfigResp {
	items := make([]settingsv1.ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return settingsv1.SystemConfigResp{Items: transport.Ptrs(items)}
}

func configItemResponse(item settingsdto.ConfigItem) settingsv1.ConfigItemResp {
	return settingsv1.ConfigItemResp{Key: item.Key, Value: transport.ProtoValue(item.Value), Default: transport.ProtoValue(item.Default), IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description}
}
