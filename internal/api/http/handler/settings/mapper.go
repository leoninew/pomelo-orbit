package settingshandler

import (
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	settingsdto "github.com/leoninew/pomelo-orbit/internal/application/settings/dto"
	settingsv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/settings"
	"google.golang.org/protobuf/types/known/structpb"
)

func configUpdates(req *settingsv1.SystemConfigUpdateReq) []settingsdto.Update {
	updates := make([]settingsdto.Update, 0, len(req.Updates))
	for _, item := range req.Updates {
		if item == nil {
			updates = append(updates, settingsdto.Update{})
		} else {
			updates = append(updates, settingsdto.Update{Key: item.Key, Value: transport.NativeValue(item.Value)})
		}
	}
	return updates
}

func systemConfigResponse(config settingsdto.SystemConfig) settingsv1.SystemConfigResp {
	items := make([]settingsv1.ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return settingsv1.SystemConfigResp{Items: transport.Ptrs(items), Revision: config.Revision, PendingRestart: config.PendingRestart, NextConfigError: config.NextConfigError}
}

func configItemResponse(item settingsdto.ConfigItem) settingsv1.ConfigItemResp {
	return settingsv1.ConfigItemResp{Key: item.Key, Value: configValue(item.Value), Default: configValue(item.Default), IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description, OverrideValue: configValue(item.OverrideValue), NextValue: configValue(item.NextValue), Type: item.Type, PendingRestart: item.PendingRestart, ValueSource: item.ValueSource, DefaultSource: item.DefaultSource, NextSource: item.NextSource, NextValueKnown: item.NextValueKnown}
}

func configValue(value any) *structpb.Value {
	if values, ok := value.([]string); ok {
		list := make([]any, len(values))
		for index, entry := range values {
			list[index] = entry
		}
		return transport.ProtoValue(list)
	}
	return transport.ProtoValue(value)
}
