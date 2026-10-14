package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestSharedMountDeclarationAndOwnership(t *testing.T) {
	joinNetwork := false
	plan := model.EffectiveServicePlan{Application: model.Application{Code: "workspace", Kind: "standard"}, JoinTraefikNetwork: &joinNetwork, Components: []model.EffectiveServiceComponent{{
		Name: "worker", Image: "worker:1", PullPolicy: "missing", Mounts: []model.VersionComponentMount{
			{SourceType: "directory", Source: "/host", Target: "/data", SourceIsHostPath: true, Shared: true},
			{SourceType: "directory", Source: "/private", Target: "/logs", SourceIsHostPath: true},
		},
	}}}
	compose, err := RenderCompose(context.Background(), RenderInput{Plan: plan})
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
		name, source, desired string
		shared, allowed       bool
	}{
		{"shared workspace", "/host", "/host/b", true, true},
		{"unmarked data", "/host", "/host/b", false, false},
		{"parent write", "/host/b/data", "/host/b", true, false},
		{"nested private source", "/host", "/host/b", true, false},
		{"Windows shared workspace", `D:\Data`, "/run/desktop/mnt/host/d/data/b", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, runtime, _ := directoryMountTestService(t)
			runtime.mounts = []deploymentport.RunningBindMount{{Source: tc.source, Shared: tc.shared}}
			if tc.name == "nested private source" {
				runtime.mounts = append(runtime.mounts, deploymentport.RunningBindMount{Source: "/host/b/data"})
			}
			err := service.checkRunningMountOwnership(context.Background(), deploymentTestTarget(1), "any-service", []string{tc.desired}, "linux")
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
			runtime.mounts = []deploymentport.RunningBindMount{{Source: "/host", Shared: true}}
			mounts := []ResolvedMount{{Relative: true, HostSource: "/host/b/data"}}
			if err := service.checkDirectoryOwnership(context.Background(), target, current, directory, mounts); err != nil {
				t.Fatalf("shared workspace rejected: %v", err)
			}
			runtime.mounts[0].Shared = false
			if err := service.checkDirectoryOwnership(context.Background(), target, current, directory, mounts); err == nil || !strings.Contains(err.Error(), "running service a") {
				t.Fatalf("unmarked workspace accepted: %v", err)
			}
		})
	}
}
