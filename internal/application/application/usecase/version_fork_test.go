package applicationsvc

import (
	"context"
	"reflect"
	"testing"

	applicationport "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestForkVersionForBuildCopiesCompleteComponentConfiguration(t *testing.T) {
	note := "source note"
	restartPolicy := "unless-stopped"
	bindAddress := "127.0.0.1"
	listenPort := 8080
	entrypoint := "web"
	pathPrefix := "/api"
	interval := "30s"
	retries := 3
	limitCPU := "2"
	artifactId := "artifact-source"
	source := model.Version{Id: "version-source", ApplicationId: "application-1", Label: "source", Status: "published", Note: &note}
	store := &fakeVersionForkStore{source: source, components: []model.VersionComponent{{
		Id: "component-source", VersionId: "version-source", Name: "web", Image: "example/web:source", ArtifactId: &artifactId,
		Command: []string{"serve", "--port", "8080"}, PullPolicy: "always", RestartPolicy: &restartPolicy,
		Env:          []model.VersionComponentEnv{{Key: "ENV", Value: "production"}},
		Endpoints:    []model.VersionComponentEndpoint{{Name: "http", Protocol: "http", ContainerPort: 8080, Mode: "gateway_http", BindAddress: &bindAddress, ListenPort: &listenPort, Entrypoint: &entrypoint, PathPrefix: &pathPrefix}},
		Mounts:       []model.VersionComponentMount{{SourceType: "controlled_file", Source: "config/app.env", Target: "/app/.env", Content: "KEY=value", Mode: "0644", IgnoreIfExists: true}},
		Dependencies: []model.VersionComponentDependency{{Name: "database", Condition: "service_healthy"}},
		Healthcheck:  &model.VersionComponentHealthcheck{TestMode: "CMD", Test: "curl -f http://localhost/health", Interval: &interval, Retries: &retries},
		Resources:    &model.VersionComponentResources{LimitCPUs: &limitCPU},
		Tmpfs:        []model.VersionComponentTmpfs{{Target: "/tmp", SizeBytes: 1024, Mode: "1777"}},
		Ulimits:      []model.VersionComponentUlimit{{Name: "nofile", Soft: 1024, Hard: 2048}},
		Devices:      []model.VersionComponentDeviceRequest{{Driver: "nvidia", Count: "1", Capabilities: []string{"gpu", "compute"}}},
	}}}
	forked, err := forkVersionForBuild(context.Background(), store, applicationport.BuildVersionForkInput{
		SourceVersionId: source.Id,
		Label:           "forked",
		ComponentName:   "web",
		Image:           "example/web:build",
		ArtifactId:      "artifact-build",
	})
	if err != nil {
		t.Fatal(err)
	}
	if forked.ApplicationId != source.ApplicationId || forked.Label != "forked" || forked.Status != status.VersionStatusUnpublished || forked.CreatedFromVersionId == nil || *forked.CreatedFromVersionId != source.Id || forked.Note == nil || *forked.Note != note {
		t.Fatalf("unexpected forked version: %+v", forked)
	}
	if len(store.createdComponents) != 1 {
		t.Fatalf("expected one copied component, got %d", len(store.createdComponents))
	}
	got := store.createdComponents[0]
	want := store.components[0]
	want.Id = got.Id
	want.VersionId = forked.Id
	want.Image = "example/web:build"
	want.ArtifactId = ptr("artifact-build")
	if got.Id == store.components[0].Id || got.VersionId != forked.Id || !reflect.DeepEqual(got, want) {
		t.Fatalf("copied component differs from source:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestForkVersionForBuildDoesNotPersistWhenTargetComponentIsMissing(t *testing.T) {
	store := &fakeVersionForkStore{
		source:     model.Version{Id: "version-source", ApplicationId: "application-1"},
		components: []model.VersionComponent{{Id: "component-source", Name: "web"}},
	}
	_, err := forkVersionForBuild(context.Background(), store, applicationport.BuildVersionForkInput{
		SourceVersionId: "version-source",
		Label:           "forked",
		ComponentName:   "worker",
		Image:           "example/worker:build",
		ArtifactId:      "artifact-build",
	})
	if err == nil {
		t.Fatal("expected mutation error")
	}
	if store.created {
		t.Fatal("fork must not persist when the build mutation fails")
	}
}

type fakeVersionForkStore struct {
	source            model.Version
	components        []model.VersionComponent
	createdVersion    model.Version
	createdComponents []model.VersionComponent
	created           bool
}

func (s *fakeVersionForkStore) Version(_ context.Context, _ string) (model.Version, error) {
	return s.source, nil
}

func (s *fakeVersionForkStore) VersionComponentsByVersion(_ context.Context, _ string) ([]model.VersionComponent, error) {
	return append([]model.VersionComponent(nil), s.components...), nil
}

func (s *fakeVersionForkStore) CreateVersionWithVersionComponents(_ context.Context, version model.Version, components []model.VersionComponent) error {
	s.created = true
	s.createdVersion = version
	s.createdComponents = append([]model.VersionComponent(nil), components...)
	return nil
}

func ptr(value string) *string {
	return &value
}
