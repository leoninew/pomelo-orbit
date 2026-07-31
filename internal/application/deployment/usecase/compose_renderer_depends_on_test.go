package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComponentPreservesDependencyConditions(t *testing.T) {
	service, _, err := renderVersionComponentService(model.VersionComponent{
		Name: "api", Image: "nginx",
		Dependencies: []model.VersionComponentDependency{
			{Name: "mysql", Condition: "service_healthy"},
			{Name: "redis", Condition: "service_started"},
		},
	}, "demo", "", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	depends, ok := service["depends_on"].(map[string]map[string]string)
	if !ok {
		t.Fatalf("depends type = %T", service["depends_on"])
	}
	if depends["mysql"]["condition"] != "service_healthy" {
		t.Fatalf("mysql condition = %#v", depends["mysql"])
	}
}

func TestRenderComposeDeclaresNamedVolumes(t *testing.T) {
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Version: model.Version{Id: "version-1"},
		Components: []model.VersionComponent{{
			Name:  "db",
			Image: "postgres:16",
			Mounts: []model.VersionComponentMount{{
				SourceType: mountSourceNamedVolume,
				Source:     "postgres-data",
				Target:     "/var/lib/postgresql/data",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compose, "volumes:\n    postgres-data: {}") {
		t.Fatalf("compose must declare named volume:\n%s", compose)
	}
}

func TestRenderComponentMergesResourceReservationsAndDevices(t *testing.T) {
	reservationCPUs := "2"
	reservationMemory := "8g"
	service, _, err := renderVersionComponentService(model.VersionComponent{
		Name: "tei", Image: "ghcr.io/huggingface/text-embeddings-inference:cuda-1.9.3",
		Resources: &model.VersionComponentResources{
			ReservationCPUs:   &reservationCPUs,
			ReservationMemory: &reservationMemory,
		},
		Devices: []model.VersionComponentDeviceRequest{{
			Driver: "nvidia", Count: "all", Capabilities: []string{"gpu"},
		}},
	}, "demo", "", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	deploy, ok := service["deploy"].(map[string]any)
	if !ok {
		t.Fatalf("deploy type = %T", service["deploy"])
	}
	resources, ok := deploy["resources"].(map[string]any)
	if !ok {
		t.Fatalf("resources type = %T", deploy["resources"])
	}
	reservations, ok := resources["reservations"].(map[string]any)
	if !ok {
		t.Fatalf("reservations type = %T", resources["reservations"])
	}
	if reservations["cpus"] != reservationCPUs || reservations["memory"] != reservationMemory {
		t.Fatalf("reservations = %#v", reservations)
	}
	devices, ok := reservations["devices"].([]map[string]any)
	if !ok || len(devices) != 1 {
		t.Fatalf("devices = %#v", reservations["devices"])
	}
	if devices[0]["driver"] != "nvidia" || devices[0]["count"] != "all" {
		t.Fatalf("device = %#v", devices[0])
	}
}
