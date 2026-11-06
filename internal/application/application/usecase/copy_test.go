package applicationsvc

import (
	"context"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type copyProjectReader struct {
	members bool
}

func (f copyProjectReader) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}

func (f copyProjectReader) IsProjectMember(context.Context, string, string) (bool, error) {
	return f.members, nil
}

type copyApplicationStore struct {
	repository.ApplicationStore
	application model.Application
	versions    []applicationdto.VersionDefinition
	nameExists  bool
	codeExists  bool
	createdApp  model.Application
	created     []applicationdto.VersionDefinition
}

func (f *copyApplicationStore) Application(_ context.Context, _, _ string) (model.Application, error) {
	return f.application, nil
}

func (f *copyApplicationStore) ListVersions(_ context.Context, _, _ string) ([]model.Version, error) {
	versions := make([]model.Version, len(f.versions))
	for index, version := range f.versions {
		versions[index] = version.Version
	}
	return versions, nil
}

func (f *copyApplicationStore) VersionComponentsByVersion(_ context.Context, _, versionId string) ([]model.VersionComponent, error) {
	for _, version := range f.versions {
		if version.Version.Id == versionId {
			return version.Components, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *copyApplicationStore) ApplicationByProjectAndName(context.Context, string, string) (model.Application, error) {
	if f.nameExists {
		return model.Application{Name: f.application.Name}, nil
	}
	return model.Application{}, repository.ErrNotFound
}

func (f *copyApplicationStore) ApplicationByProjectAndCode(context.Context, string, string) (model.Application, error) {
	if f.codeExists {
		return model.Application{Code: f.application.Code}, nil
	}
	return model.Application{}, repository.ErrNotFound
}

func (f *copyApplicationStore) CreateApplication(_ context.Context, app model.Application) error {
	f.createdApp = app
	return nil
}

func (f *copyApplicationStore) CreateVersionWithVersionComponents(_ context.Context, _ string, version model.Version, components []model.VersionComponent) error {
	f.created = append(f.created, applicationdto.VersionDefinition{Version: version, Components: components})
	return nil
}

func TestCopyApplicationUsesSelectedVersionScopeAndRemapsLineage(t *testing.T) {
	parentId := "version-parent"
	artifactId := "artifact-source"
	latestNote := "latest note"
	versions := []applicationdto.VersionDefinition{
		{
			Version: model.Version{Id: "version-latest", Label: "release-2", Status: "published", CreatedFromVersionId: &parentId, Note: &latestNote},
			Components: []model.VersionComponent{{
				Id: "component-latest", VersionId: "version-latest", Name: "db", Image: "postgres:18", ArtifactId: &artifactId,
				Env:    []model.VersionComponentEnv{{Key: "PORT", Value: "5432"}},
				Mounts: []model.VersionComponentMount{{Source: `C:\data`, Target: "/var/lib/postgresql/data", Content: "kept"}},
			}},
		},
		{
			Version:    model.Version{Id: parentId, Label: "release-1", Status: "published"},
			Components: []model.VersionComponent{{Id: "component-parent", VersionId: parentId, Name: "db", Image: "postgres:17"}},
		},
	}

	for _, test := range []struct {
		name       string
		all        bool
		wantCount  int
		wantParent bool
	}{
		{name: "latest", wantCount: 1},
		{name: "all", all: true, wantCount: 2, wantParent: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &copyApplicationStore{
				application: model.Application{Id: "application-source", Name: "Postgres", Code: "postgres", Kind: "standard"},
				versions:    versions,
			}
			service := New(copyProjectReader{members: true}, store)
			created, err := service.CopyApplication(context.Background(), "user-1", "source", "target", "application-source", test.all)
			if err != nil {
				t.Fatal(err)
			}
			if created.Application.Id == "application-source" || created.Application.ProjectId == nil || *created.Application.ProjectId != "target" {
				t.Fatalf("copied application = %+v", created.Application)
			}
			if len(store.created) != test.wantCount {
				t.Fatalf("created versions = %d, want %d", len(store.created), test.wantCount)
			}
			var latest applicationdto.VersionDefinition
			for _, version := range store.created {
				if version.Version.Label == "release-2" {
					latest = version
				}
			}
			if latest.Version.Id == "" {
				t.Fatal("latest version was not created")
			}
			if latest.Version.Id != created.LatestVersionId {
				t.Fatalf("latest version id = %q, want %q", latest.Version.Id, created.LatestVersionId)
			}
			if test.wantParent {
				if latest.Version.CreatedFromVersionId == nil || *latest.Version.CreatedFromVersionId != created.VersionIds[parentId] {
					t.Fatalf("latest parent = %v, want mapped version %q", latest.Version.CreatedFromVersionId, created.VersionIds[parentId])
				}
			} else if latest.Version.CreatedFromVersionId != nil {
				t.Fatalf("latest-only copy retained source parent %q", *latest.Version.CreatedFromVersionId)
			}
			component := latest.Components[0]
			if component.Id == "component-latest" || component.VersionId != latest.Version.Id || component.ArtifactId == nil || *component.ArtifactId != artifactId {
				t.Fatalf("copied component identity/artifact = %+v", component)
			}
			if component.Mounts[0].Source != `C:\data` || component.Mounts[0].Content != "kept" {
				t.Fatalf("component mount changed: %+v", component.Mounts[0])
			}
		})
	}
}

func TestCopyApplicationRejectsTargetNameConflictBeforeWrites(t *testing.T) {
	store := &copyApplicationStore{
		application: model.Application{Id: "application-source", Name: "Postgres", Code: "postgres", Kind: "standard"},
		versions: []applicationdto.VersionDefinition{{
			Version: model.Version{Id: "version-latest", Label: "release-1", Status: "published"},
		}},
		nameExists: true,
	}
	service := New(copyProjectReader{members: true}, store)
	_, err := service.CopyApplication(context.Background(), "user-1", "source", "target", "application-source", false)
	if !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("copy error = %v, want conflict", err)
	}
	if store.createdApp.Id != "" || len(store.created) != 0 {
		t.Fatalf("copy wrote resources before conflict: app=%+v versions=%d", store.createdApp, len(store.created))
	}
}
