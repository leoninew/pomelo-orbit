package repositoryhandler

import (
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
)

func variableDeclarationResponses(items []map[string]any) []commonv1.VariableDeclarationResp {
	resp := make([]commonv1.VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, variableDeclarationResponse(item))
	}
	return resp
}

func variableDeclarationResponse(item map[string]any) commonv1.VariableDeclarationResp {
	return commonv1.VariableDeclarationResp{
		Name:        stringFromMap(item, "name"),
		Description: stringFromMap(item, "description"),
		Default:     transportresponse.ProtoValue(item["default"]),
		Value:       transportresponse.ProtoValue(item["value"]),
		Secret:      boolFromMap(item, "secret"),
		Source:      stringFromMap(item, "source"),
		Editable:    boolFromMap(item, "editable"),
		StageId:     stringFromMap(item, "stage_id"),
		StageName:   stringFromMap(item, "stage_name"),
	}
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
		"default":     transportresponse.NativeValue(item.Default),
		"value":       transportresponse.NativeValue(item.Value),
		"secret":      item.Secret,
		"source":      item.Source,
		"editable":    item.Editable,
		"stage_id":    item.StageId,
	}
}

func stringFromMap(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return value
}

func boolFromMap(item map[string]any, key string) bool {
	value, _ := item[key].(bool)
	return value
}
