package settingshandler

import (
	"reflect"
	"testing"

	settingsdto "github.com/leoninew/pomelo-orbit/internal/application/settings/dto"
	settingsv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/settings"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConfigurationValuesPreserveListsAndExplicitEmptyValues(t *testing.T) {
	for _, value := range []any{[]string{}, []string{"https://example.com"}, "", false, 0} {
		response := configItemResponse(settingsdto.ConfigItem{Value: value, Default: value, OverrideValue: value, NextValue: value, IsOverridden: true, NextValueKnown: true})
		for _, converted := range []*structpb.Value{response.Value, response.Default, response.OverrideValue, response.NextValue} {
			if converted.GetKind() == nil || reflect.TypeOf(converted.GetKind()) == reflect.TypeFor[*structpb.Value_NullValue]() {
				t.Fatalf("value %#v became null", value)
			}
		}
	}
	response := configItemResponse(settingsdto.ConfigItem{Value: "current", NextValueKnown: false})
	if response.OverrideValue.AsInterface() != nil || response.IsOverridden {
		t.Fatal("missing override acquired a value")
	}
}

func TestBatchMappingDoesNotSilentlyDropNullUpdates(t *testing.T) {
	updates := configUpdates(&settingsv1.SystemConfigUpdateReq{Updates: []*settingsv1.ConfigUpdateItem{{Key: "app__debug", Value: structpb.NewBoolValue(true)}, nil}})
	if len(updates) != 2 || updates[1].Key != "" {
		t.Fatal("null update was silently omitted")
	}
}
