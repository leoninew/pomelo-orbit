package servicesvc

import (
	"context"
	"reflect"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type copyApplicationFake struct {
	result applicationdto.ApplicationCopyResult
}

type copyEnvironmentFake struct{ view environmentdto.View }

func (f copyEnvironmentFake) EnvironmentForUser(context.Context, string, string) (environmentdto.View, error) {
	return f.view, nil
}

func (f copyApplicationFake) CopyApplication(context.Context, string, string, string, string, bool) (applicationdto.ApplicationCopyResult, error) {
	return f.result, nil
}

type copyServiceStore struct {
	repository.ServiceStore
	source           model.Service
	components       []model.ServiceComponent
	env              []model.ServiceEnv
	exists           bool
	created          model.Service
	copiedComponents []model.ServiceComponent
	copiedEnv        []model.ServiceEnv
	directory        string
}

func (f *copyServiceStore) Service(context.Context, string, string) (model.Service, error) {
	return f.source, nil
}
func (f *copyServiceStore) ServiceComponentsByService(context.Context, string, string) ([]model.ServiceComponent, error) {
	return f.components, nil
}
func (f *copyServiceStore) ServiceEnvByService(context.Context, string, string) ([]model.ServiceEnv, error) {
	return f.env, nil
}
func (f *copyServiceStore) ServiceByProjectAndCode(context.Context, string, string) (model.Service, error) {
	if f.exists {
		return model.Service{Id: "existing"}, nil
	}
	return model.Service{}, repository.ErrNotFound
}
func (f *copyServiceStore) CreateServiceWithComponents(_ context.Context, _ string, service model.Service, components []model.ServiceComponent) error {
	f.created, f.copiedComponents = service, components
	return nil
}
func (f *copyServiceStore) ReplaceServiceEnv(_ context.Context, _, _ string, env []model.ServiceEnv) error {
	f.copiedEnv = env
	return nil
}
func (f *copyServiceStore) UpdateServiceDeploymentDirectory(_ context.Context, _, _, directory string, _ int64) error {
	f.directory = directory
	return nil
}

func TestCopyServicePreservesRawConfigurationAndSelectsVersion(t *testing.T) {
	for _, all := range []bool{false, true} {
		t.Run(map[bool]string{false: "latest", true: "all"}[all], func(t *testing.T) {
			store := &copyServiceStore{
				source:     model.Service{Id: "source-service", ProjectId: "source", ApplicationId: "source-app", VersionId: "old-version", Code: "db-default", Status: "running", DeploymentDirectory: `E:\orbit\db`, DirectoryTargetRevision: 9, RuntimeDirectory: "running-dir", RuntimeTargetRevision: 9, CurrentDeploymentId: stringPtr("deployment")},
				env:        []model.ServiceEnv{{Key: "PORT", Value: "5432"}},
				components: []model.ServiceComponent{{Id: "source-component", SourceVersionComponentId: "old-declaration", ServiceId: "source-service", ComponentName: "db", Status: "active", Entrypoint: []string{}, Command: nil, User: stringPtr(""), Env: []model.ServiceComponentEnv{{Key: "PORT", Value: stringPtr("5432"), State: model.ServiceComponentOverlayOverride}}, Mounts: []model.ServiceComponentMount{{Target: "/data", State: model.ServiceComponentOverlayDeleted}}}},
			}
			copier := NewCopyService(New(serviceCreateProjectFake{}, serviceApplicationFake{declarations: []model.VersionComponent{{Id: "new-latest-declaration", Name: "db"}}}, store, nil), copyApplicationFake{result: applicationdto.ApplicationCopyResult{
				Application: model.Application{Id: "new-app"}, LatestVersionId: "new-latest-version",
				VersionIds: map[string]string{"old-version": "new-old-version"}, VersionComponentIds: map[string]string{"old-declaration": "new-old-declaration"},
			}}, copyEnvironmentFake{view: environmentdto.View{TargetType: "ssh", TargetRevision: 2, SSH: &environmentdto.SSHTargetView{Platform: "linux"}}})
			copied, err := copier.Copy(context.Background(), "user", "source", "source-service", servicedto.ServiceCopyInput{
				TargetProjectId: "target", CopyAllVersions: all, DeploymentDirectory: "/custom/db", EnvironmentTargetRevision: 2,
			})
			if err != nil {
				t.Fatal(err)
			}
			wantVersion, wantDeclaration := "new-latest-version", "new-latest-declaration"
			if all {
				wantVersion, wantDeclaration = "new-old-version", "new-old-declaration"
			}
			if copied.Id == store.source.Id || copied.ProjectId != "target" || copied.ApplicationId != "new-app" || copied.VersionId != wantVersion || copied.Code != store.source.Code {
				t.Fatal("copied service identity or version is incorrect")
			}
			if copied.Status != "stopped" || copied.CurrentDeploymentId != nil || copied.RuntimeDirectory != "" || copied.DirectoryTargetRevision != 2 || copied.RuntimeTargetRevision != 0 || store.directory != "/custom/db" {
				t.Fatal("copy retained runtime state or lost target deployment path")
			}
			component := store.copiedComponents[0]
			if component.Id == "source-component" || component.ServiceId != copied.Id || component.SourceVersionComponentId != wantDeclaration {
				t.Fatal("component references were not remapped")
			}
			component.Id, component.ServiceId, component.SourceVersionComponentId = "source-component", "source-service", "old-declaration"
			if !reflect.DeepEqual(component, store.components[0]) || !reflect.DeepEqual(store.copiedEnv, store.env) {
				t.Fatal("copy changed raw configuration")
			}
			if store.source.Status != "running" || store.components[0].SourceVersionComponentId != "old-declaration" {
				t.Fatal("source was modified")
			}
		})
	}
}

func TestCopyServiceRejectsConflictAndLatestTopologyChange(t *testing.T) {
	for _, test := range []struct {
		name         string
		exists       bool
		declarations []model.VersionComponent
	}{
		{name: "service code", exists: true, declarations: []model.VersionComponent{{Id: "new-component", Name: "db"}}},
		{name: "latest topology", declarations: []model.VersionComponent{{Id: "new-component", Name: "api"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &copyServiceStore{source: model.Service{Id: "source-service", ApplicationId: "source-app", VersionId: "old-version", Code: "db-default"}, exists: test.exists, components: []model.ServiceComponent{{Id: "source-component", ComponentName: "db", SourceVersionComponentId: "old-component"}}}
			copier := NewCopyService(New(serviceCreateProjectFake{}, serviceApplicationFake{declarations: test.declarations}, store, nil), copyApplicationFake{result: applicationdto.ApplicationCopyResult{LatestVersionId: "latest"}}, copyEnvironmentFake{view: environmentdto.View{TargetType: "ssh", TargetRevision: 2, SSH: &environmentdto.SSHTargetView{Platform: "linux"}}})
			_, err := copier.Copy(context.Background(), "user", "source", "source-service", servicedto.ServiceCopyInput{
				TargetProjectId: "target", DeploymentDirectory: "/custom/db", EnvironmentTargetRevision: 2,
			})
			if !apperror.IsKind(err, apperror.KindConflict) || store.created.Id != "" {
				t.Fatalf("expected conflict before service writes, got %v", err)
			}
		})
	}
}

func TestCopyServiceValidatesTargetDirectoryBeforeCopyingApplication(t *testing.T) {
	for _, test := range []struct {
		name, directory string
		revision        int64
		kind            apperror.Kind
	}{
		{name: "Windows source path on Linux target", directory: `E:\orbit\db`, revision: 2, kind: apperror.KindValidation},
		{name: "missing directory", revision: 2, kind: apperror.KindValidation},
		{name: "target changed", directory: "/custom/db", revision: 1, kind: apperror.KindConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &copyServiceStore{source: model.Service{Id: "source-service"}}
			copier := NewCopyService(New(serviceCreateProjectFake{}, nil, store, nil), nil, copyEnvironmentFake{view: environmentdto.View{TargetType: "ssh", TargetRevision: 2, SSH: &environmentdto.SSHTargetView{Platform: "linux"}}})
			_, err := copier.Copy(context.Background(), "user", "source", "source-service", servicedto.ServiceCopyInput{
				TargetProjectId: "target", DeploymentDirectory: test.directory, EnvironmentTargetRevision: test.revision,
			})
			if !apperror.IsKind(err, test.kind) || store.created.Id != "" {
				t.Fatalf("expected directory rejection %v before writes, got %v", test.kind, err)
			}
		})
	}
}
