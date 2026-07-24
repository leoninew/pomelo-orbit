package repositoryhandler

import (
	"testing"

	commonv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/common"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestVariableDeclarationRequestMapsSkipsNilAndPreservesDynamicValues(t *testing.T) {
	value, err := structpb.NewValue(map[string]any{"region": "cn"})
	if err != nil {
		t.Fatalf("create dynamic value: %v", err)
	}
	items := variableDeclarationRequestMaps([]*commonv1.VariableDeclarationReq{nil, {Name: "deploy", Value: value, Secret: true}})
	if len(items) != 1 || items[0]["name"] != "deploy" || items[0]["secret"] != true {
		t.Fatalf("unexpected variable declarations: %#v", items)
	}
	actual, ok := items[0]["value"].(map[string]any)
	if !ok || actual["region"] != "cn" {
		t.Fatalf("dynamic value was not preserved: %#v", items[0]["value"])
	}
}

func TestVariableDeclarationResponsesKeepsEmptySlice(t *testing.T) {
	responses := variableDeclarationResponses(nil)
	if responses == nil || len(responses) != 0 {
		t.Fatalf("nil source must map to a non-nil empty response slice: %#v", responses)
	}
}
