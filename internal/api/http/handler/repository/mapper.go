package repositoryhandler

import (
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	"github.com/leoninew/pomelo-orbit/internal/application/variableview"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
)

func variableResponses(items []variableview.View) []*commonv1.VariableResp {
	resp := make([]*commonv1.VariableResp, 0, len(items))
	for _, item := range items {
		var binding *commonv1.VariableStageBindingResp
		if item.StageBinding != nil {
			binding = &commonv1.VariableStageBindingResp{StageId: item.StageBinding.StageId, StageName: item.StageBinding.StageName}
		}
		resp = append(resp, &commonv1.VariableResp{Name: item.Name, Kind: string(item.Kind), Scope: item.Scope, StageBinding: binding, Configuration: variableConfigurationResponse(item.Configuration), GlobalConfiguration: variableConfigurationResponse(item.GlobalConfiguration), StageOverride: variableConfigurationResponse(item.StageOverride), Editable: item.Editable})
	}
	return resp
}

func variableConfigurationResponse(item *variableview.Configuration) *commonv1.VariableConfigurationResp {
	if item == nil {
		return nil
	}
	return &commonv1.VariableConfigurationResp{Description: item.Description, Default: transport.ProtoValue(item.Default), Value: transport.ProtoValue(item.Value), Secret: item.Secret, Editable: item.Editable}
}

func variableDeclarationRequestMaps(items []*commonv1.VariableDeclarationReq) []map[string]any {
	resp := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, variableDeclarationRequestMap(item))
	}
	return resp
}

func variableDeclarationRequestMap(item *commonv1.VariableDeclarationReq) map[string]any {
	return map[string]any{
		"name":        item.Name,
		"description": item.Description,
		"default":     transport.NativeValue(item.Default),
		"value":       transport.NativeValue(item.Value),
		"secret":      item.Secret,
		"source":      item.Source,
		"editable":    item.Editable,
		"stage_id":    item.StageId,
	}
}
