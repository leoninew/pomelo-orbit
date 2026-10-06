package deploymentsvc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestSharedMountLabelMatchesRunningMount(t *testing.T) {
	joinNetwork := false
	plan := model.EffectiveServicePlan{Application: model.Application{Code: "workspace", Kind: "standard"}, JoinTraefikNetwork: &joinNetwork, Components: []model.EffectiveServiceComponent{{
		Name: "worker", Image: "worker:1", PullPolicy: "missing", Mounts: []model.VersionComponentMount{
			{SourceType: "directory", Source: "/host", Target: "/data", SourceIsHostPath: true, Shared: true},
			{SourceType: "directory", Source: "/private", Target: "/logs", SourceIsHostPath: true},
		},
	}}}
	compose, err := (Service{}).RenderCompose(context.Background(), RenderInput{Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Services map[string]struct{ Labels []string }
	}
	if err := yaml.Unmarshal([]byte(compose), &parsed); err != nil {
		t.Fatal(err)
	}
	var label string
	for _, value := range parsed.Services["worker"].Labels {
		if strings.HasPrefix(value, sharedMountsLabel+"=") {
			label = strings.TrimPrefix(value, sharedMountsLabel+"=")
		}
	}
	if label != `["/data"]` {
		t.Fatalf("shared label=%q", label)
	}
	sharedHash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Components[0].Mounts[0].Shared = false
	exclusiveHash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	if sharedHash == exclusiveHash {
		t.Fatal("shared declaration did not affect the configuration hash")
	}
	for _, tc := range []struct {
		name, label, source, destination, desired string
		allowed                                   bool
	}{
		{"shared workspace", label, "/host", "/data", "/host/b", true},
		{"normalized target", `["/data/"]`, "/host", "/data", "/host/b", true},
		{"unmarked data", "", "/host", "/data", "/host/b", false},
		{"different target", label, "/host", "/logs", "/host/b", false},
		{"nested private source", label, "/host/b/data", "/data/private", "/host/b", false},
		{"parent write", label, "/host/b/data", "/data", "/host/b", false},
		{"Windows shared workspace", label, `D:\Data`, "/data", "/run/desktop/mnt/host/d/data/b", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, runtime, _ := directoryMountTestService(t)
			runtime.psOutput = "any-managed-container"
			fixture := []map[string]any{{
				"State":  map[string]bool{"Running": true},
				"Config": map[string]any{"Labels": map[string]string{sharedMountsLabel: tc.label}},
				"Mounts": []map[string]string{{"Type": "bind", "Source": tc.source, "Destination": tc.destination}},
			}}
			if tc.name == "nested private source" {
				fixture[0]["Mounts"] = append(fixture[0]["Mounts"].([]map[string]string), map[string]string{"Type": "bind", "Source": "/host", "Destination": "/data"})
			}
			encoded, err := json.Marshal(fixture)
			if err != nil {
				t.Fatal(err)
			}
			runtime.inspectOutput = string(encoded)
			err = service.checkRunningMountOwnership(context.Background(), deploymentTestTarget(1), "any-service", []string{tc.desired}, "linux")
			if (err == nil) != tc.allowed {
				t.Fatalf("allowed=%t, err=%v", tc.allowed, err)
			}
		})
	}
}

func TestSharedMountDoesNotReleaseServiceRoot(t *testing.T) {
	service, runtime, current := directoryMountTestService(t)
	service.executionStore.(*directoryOwnershipStore).services[1].RuntimeDirectory = "/app"
	if err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", nil); err == nil || !strings.Contains(err.Error(), "deployment directory overlaps") {
		t.Fatalf("root protection lost: %v", err)
	}
	if len(runtime.queries) != 0 {
		t.Fatal("root conflict should be rejected before inspecting shared mounts")
	}
}

func TestSharedWorkspaceOwnershipUsesLocalAndSSHPaths(t *testing.T) {
	for _, targetType := range []string{model.EnvironmentTargetTypeLocal, model.EnvironmentTargetTypeSSH} {
		t.Run(targetType, func(t *testing.T) {
			service, runtime, current := directoryMountTestService(t)
			target := deploymentTestTarget(1)
			target.Environment.TargetType = targetType
			directory := "/app/b"
			if targetType == model.EnvironmentTargetTypeLocal {
				target.Environment.SSH = nil
			} else {
				target.Environment.SSH.Platform = model.EnvironmentPlatformLinux
				directory = "/host/b"
			}
			runtime.psOutput = "shared-workspace-container"
			fixture := []map[string]any{{
				"State":  map[string]bool{"Running": true},
				"Config": map[string]any{"Labels": map[string]string{sharedMountsLabel: `["/data"]`}},
				"Mounts": []map[string]string{{"Type": "bind", "Source": "/host", "Destination": "/data"}},
			}}
			encoded, err := json.Marshal(fixture)
			if err != nil {
				t.Fatal(err)
			}
			runtime.inspectOutput = string(encoded)
			mounts := []ResolvedMount{{Relative: true, HostSource: "/host/b/data"}}
			if err := service.checkDirectoryOwnership(context.Background(), target, current, directory, mounts); err != nil {
				t.Fatalf("shared workspace rejected: %v", err)
			}
			fixture[0]["Config"] = map[string]any{"Labels": map[string]string{}}
			encoded, err = json.Marshal(fixture)
			if err != nil {
				t.Fatal(err)
			}
			runtime.inspectOutput = string(encoded)
			if err := service.checkDirectoryOwnership(context.Background(), target, current, directory, mounts); err == nil || !strings.Contains(err.Error(), "running service a") {
				t.Fatalf("unmarked workspace accepted: %v", err)
			}
		})
	}
}
