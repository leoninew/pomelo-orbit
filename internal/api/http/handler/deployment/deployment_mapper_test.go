package deploymenthandler

import (
	"encoding/json"
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestApplicationStatusResponse(t *testing.T) {
	response := applicationStatusResponse([]deploymentdto.RuntimeContainer{{
		Id:           "container-1",
		Name:         "demo-web-1",
		Service:      "web",
		State:        "running",
		Status:       "Up 1 minute",
		Health:       "healthy",
		Image:        "nginx:latest",
		VersionId:    "version-1",
		VersionLabel: "2026.07.28",
		ComponentId:  "component-1",
	}})

	if len(response.Containers) != 1 {
		t.Fatalf("container count: got %d want 1", len(response.Containers))
	}
	container := response.Containers[0]
	if container.Id != "container-1" || container.Name != "demo-web-1" || container.Service != "web" {
		t.Fatalf("identity fields: got %+v", container)
	}
	if container.State != "running" || container.Status != "Up 1 minute" || container.Health != "healthy" || container.Image != "nginx:latest" {
		t.Fatalf("runtime fields: got %+v", container)
	}
	if container.VersionId != "version-1" || container.VersionLabel != "2026.07.28" {
		t.Fatalf("version fields: got %+v", container)
	}
	if container.ComponentId != "component-1" {
		t.Fatalf("component ID: got %+v", container)
	}

	encoded, err := protojson.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var payload struct {
		Containers []json.RawMessage `json:"containers"`
		Status     *json.RawMessage  `json:"status"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
	if len(payload.Containers) != 1 || payload.Status != nil {
		t.Fatalf("expected structured containers only, got %s", encoded)
	}
}

func TestDeploymentResponseIncludesServiceIdentity(t *testing.T) {
	serviceID := "service-1"
	instanceKey := "production"
	response := deploymentResponse(model.Deployment{Id: "deployment-1", ServiceId: &serviceID, ServiceInstanceKey: &instanceKey})

	if response.ServiceId == nil || *response.ServiceId != serviceID {
		t.Fatalf("ServiceId = %v, want %q", response.ServiceId, serviceID)
	}
	if response.ServiceInstanceKey == nil || *response.ServiceInstanceKey != instanceKey {
		t.Fatalf("ServiceInstanceKey = %v, want %q", response.ServiceInstanceKey, instanceKey)
	}
}
