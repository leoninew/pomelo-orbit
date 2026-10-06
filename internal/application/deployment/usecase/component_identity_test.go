package deploymentsvc

import (
	"context"
	"reflect"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestComponentIdentityMergeRenderAndHash(t *testing.T) {
	user, override, empty := "1000:1000", "2000:2000", ""
	joinNetwork := false
	base := model.VersionComponent{Name: "orbit", Image: "orbit:1", User: &user, GroupAdd: []string{"988"}, PullPolicy: "missing"}
	var hashes []string
	for _, tc := range []struct {
		name    string
		overlay model.ServiceComponent
		user    string
		groups  []string
	}{
		{"inherit", model.ServiceComponent{}, user, []string{"988"}},
		{"replace", model.ServiceComponent{User: &override, GroupAdd: []string{"999", "docker"}}, override, []string{"999", "docker"}},
		{"clear", model.ServiceComponent{User: &empty, GroupAdd: []string{}}, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			component, err := MergeServiceComponent(base, tc.overlay, nil)
			if err != nil {
				t.Fatal(err)
			}
			plan := model.EffectiveServicePlan{Application: model.Application{Code: "orbit", Kind: "standard"}, Components: []model.EffectiveServiceComponent{component}, JoinTraefikNetwork: &joinNetwork}
			hash, err := EffectiveServicePlanHash(plan)
			if err != nil {
				t.Fatal(err)
			}
			hashes = append(hashes, hash)
			result, err := (Service{}).RenderCompose(context.Background(), RenderInput{Plan: plan})
			if err != nil {
				t.Fatal(err)
			}
			var compose struct {
				Services map[string]struct {
					User     string
					GroupAdd []string `yaml:"group_add"`
				}
			}
			if err := yaml.Unmarshal([]byte(result), &compose); err != nil {
				t.Fatal(err)
			}
			service := compose.Services["orbit"]
			if service.User != tc.user || !reflect.DeepEqual(service.GroupAdd, tc.groups) {
				t.Fatalf("rendered identity: %+v", service)
			}
		})
	}
	if hashes[0] == hashes[1] || hashes[0] == hashes[2] {
		t.Fatal("identity changes did not affect the hash")
	}
	plan := model.EffectiveServicePlan{Components: []model.EffectiveServiceComponent{{User: &empty, GroupAdd: []string{}}}}
	emptyHash, _ := EffectiveServicePlanHash(plan)
	plan.Components[0].User, plan.Components[0].GroupAdd = nil, nil
	nilHash, _ := EffectiveServicePlanHash(plan)
	if nilHash != emptyHash {
		t.Fatal("equivalent image defaults produced different hashes")
	}
}
