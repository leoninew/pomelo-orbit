package deploymenthandler

import (
	"encoding/json"
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestApplicationStatusResponse(t *testing.T) {
	response := applicationStatusResponse([]deploymentdto.RuntimeContainer{{
		ID:      "container-1",
		Name:    "demo-web-1",
		Service: "web",
		State:   "running",
		Status:  "Up 1 minute",
		Health:  "healthy",
		Image:   "nginx:latest",
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
