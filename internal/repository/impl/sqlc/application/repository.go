package applicationrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	applicationsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/application"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.ApplicationStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *applicationsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *applicationsqlc.Queries {
		return applicationsqlc.New(dbtx)
	})
}

func projectScopeId(projectId string) sql.NullString {
	return sql.NullString{String: strings.TrimSpace(projectId), Valid: true}
}

func (r Repository) ListApplications(ctx context.Context, projectId string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	kindFilter := strings.TrimSpace(kind)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	kindValue := sql.NullString{String: kindFilter, Valid: kindFilter != ""}
	q := r.q(ctx)
	total, err := q.CountApplications(ctx, applicationsqlc.CountApplicationsParams{
		ProjectId:     projectScopeId(projectId),
		SearchPattern: searchPattern,
		Kind:          kindValue,
	})
	if err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("count applications: %w", err)
	}
	rows, err := q.ListApplications(ctx, applicationsqlc.ListApplicationsParams{
		ProjectId:     projectScopeId(projectId),
		SearchPattern: searchPattern,
		Kind:          kindValue,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("list applications: %w", err)
	}
	items := make([]model.Application, 0, len(rows))
	for _, row := range rows {
		items = append(items, appFrom(row.Id, row.ProjectId, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.Application]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) CountApplicationsByProject(ctx context.Context, projectId string) (int, error) {
	count, err := r.q(ctx).CountApplicationsByProject(ctx, projectScopeId(projectId))
	if err != nil {
		return 0, fmt.Errorf("count Project applications %s: %w", projectId, err)
	}
	return int(count), nil
}

func (r Repository) Application(ctx context.Context, projectId string, id string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationById(ctx, applicationsqlc.ApplicationByIdParams{Id: id, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return model.Application{}, fmt.Errorf("load application %s: %w", id, sqlcommon.TranslateError(err))
	}
	return appFrom(row.Id, row.ProjectId, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByName(ctx context.Context, projectId string, name string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByName(ctx, applicationsqlc.ApplicationByNameParams{Name: name, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return appFrom(row.Id, row.ProjectId, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByProjectAndName(ctx context.Context, projectId string, name string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByProjectAndName(ctx, applicationsqlc.ApplicationByProjectAndNameParams{
		ProjectId: projectScopeId(projectId),
		Name:      name,
	})
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by project and name %s/%s: %w", projectId, name, sqlcommon.TranslateError(err))
	}
	return appFrom(row.Id, row.ProjectId, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByProjectAndCode(ctx context.Context, projectId string, code string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByProjectAndCode(ctx, applicationsqlc.ApplicationByProjectAndCodeParams{ProjectId: projectScopeId(projectId), Code: code})
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by project and code %s/%s: %w", projectId, code, sqlcommon.TranslateError(err))
	}
	return appFrom(row.Id, row.ProjectId, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByCode(ctx context.Context, projectId string, code string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByCode(ctx, applicationsqlc.ApplicationByCodeParams{Code: code, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return appFrom(row.Id, row.ProjectId, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) CreateApplication(ctx context.Context, app model.Application) error {
	now := time.Now().UTC()
	createdAt, updatedAt := app.CreatedAt, app.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	err := r.q(ctx).CreateApplication(ctx, applicationsqlc.CreateApplicationParams{
		Id:        app.Id,
		ProjectId: dbmodel.NullString(app.ProjectId),
		Name:      app.Name,
		Code:      app.Code,
		Kind:      app.Kind,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})
	if err != nil {
		return fmt.Errorf("create application %s: %w", app.Name, err)
	}
	return nil
}

func (r Repository) UpdateApplication(ctx context.Context, projectId string, app model.Application) error {
	err := r.q(ctx).UpdateApplication(ctx, applicationsqlc.UpdateApplicationParams{
		Name:      app.Name,
		Code:      app.Code,
		UpdatedAt: time.Now().UTC(),
		Id:        app.Id,
		ProjectId: projectScopeId(projectId),
	})
	if err != nil {
		return fmt.Errorf("update application %s: %w", app.Id, err)
	}
	return nil
}

func (r Repository) DeleteApplication(ctx context.Context, projectId string, id string) error {
	q := r.q(ctx)
	versionIds, err := q.VersionIdsByApplication(ctx, applicationsqlc.VersionIdsByApplicationParams{ApplicationId: id, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return fmt.Errorf("list application version ids %s: %w", id, err)
	}
	for _, versionId := range versionIds {
		if err := q.ClearVersionForkRefs(ctx, applicationsqlc.ClearVersionForkRefsParams{VersionId: projectScopeId(versionId), ProjectId: projectScopeId(projectId)}); err != nil {
			return fmt.Errorf("clear version fork references %s: %w", versionId, err)
		}
	}
	if err := q.DeleteVersionsByApplication(ctx, applicationsqlc.DeleteVersionsByApplicationParams{ApplicationId: id, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("delete application versions %s: %w", id, err)
	}
	if err := q.DeleteApplication(ctx, applicationsqlc.DeleteApplicationParams{Id: id, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("delete application %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListVersions(ctx context.Context, projectId string, applicationId string) ([]model.Version, error) {
	rows, err := r.q(ctx).ListVersions(ctx, applicationsqlc.ListVersionsParams{ApplicationId: applicationId, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return nil, fmt.Errorf("list versions %s: %w", applicationId, err)
	}
	items := make([]model.Version, 0, len(rows))
	for _, row := range rows {
		items = append(items, versionFrom(row))
	}
	return items, nil
}

func (r Repository) LatestVersionByApplication(ctx context.Context, projectId string, applicationId string) (model.Version, error) {
	row, err := r.q(ctx).LatestVersionByApplication(ctx, applicationsqlc.LatestVersionByApplicationParams{ApplicationId: applicationId, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return model.Version{}, fmt.Errorf("load latest version for application %s: %w", applicationId, sqlcommon.TranslateError(err))
	}
	return versionFrom(row), nil
}

func (r Repository) ListVersionsPage(ctx context.Context, projectId string, applicationId string, page int, perPage int, search string) (repository.Page[model.Version], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	q := r.q(ctx)
	total, err := q.CountVersions(ctx, applicationsqlc.CountVersionsParams{
		ApplicationId: applicationId,
		ProjectId:     projectScopeId(projectId),
		SearchPattern: searchPattern,
	})
	if err != nil {
		return repository.Page[model.Version]{}, fmt.Errorf("count versions: %w", err)
	}
	rows, err := q.ListVersionsPage(ctx, applicationsqlc.ListVersionsPageParams{
		ApplicationId: applicationId,
		ProjectId:     projectScopeId(projectId),
		SearchPattern: searchPattern,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Version]{}, fmt.Errorf("list versions page: %w", err)
	}
	items := make([]model.Version, 0, len(rows))
	for _, row := range rows {
		items = append(items, versionFrom(row))
	}
	return repository.Page[model.Version]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Version(ctx context.Context, projectId string, id string) (model.Version, error) {
	row, err := r.q(ctx).VersionById(ctx, applicationsqlc.VersionByIdParams{Id: id, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return model.Version{}, fmt.Errorf("load version %s: %w", id, sqlcommon.TranslateError(err))
	}
	return versionFrom(row), nil
}

func (r Repository) CreateVersion(ctx context.Context, projectId string, version model.Version) error {
	if _, err := r.Application(ctx, projectId, version.ApplicationId); err != nil {
		return err
	}
	now := time.Now().UTC()
	createdAt, updatedAt := version.CreatedAt, version.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	err := r.q(ctx).CreateVersion(ctx, applicationsqlc.CreateVersionParams{
		Id:                   version.Id,
		ApplicationId:        version.ApplicationId,
		Label:                version.Label,
		Status:               version.Status,
		CreatedFromVersionId: dbmodel.NullString(version.CreatedFromVersionId),
		Note:                 dbmodel.NullString(version.Note),
		ComponentSummary:     version.ComponentSummary,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	})
	if err != nil {
		return fmt.Errorf("create version %s: %w", version.Id, err)
	}
	return nil
}

func (r Repository) UpdateVersion(ctx context.Context, projectId string, version model.Version) error {
	err := r.q(ctx).UpdateVersion(ctx, applicationsqlc.UpdateVersionParams{
		Label:            version.Label,
		Status:           version.Status,
		Note:             dbmodel.NullString(version.Note),
		ComponentSummary: version.ComponentSummary,
		UpdatedAt:        time.Now().UTC(),
		Id:               version.Id,
		ProjectId:        projectScopeId(projectId),
	})
	if err != nil {
		return fmt.Errorf("update version %s: %w", version.Id, err)
	}
	return nil
}

func (r Repository) DeleteVersion(ctx context.Context, projectId string, id string) error {
	q := r.q(ctx)
	if err := q.ClearVersionForkRefs(ctx, applicationsqlc.ClearVersionForkRefsParams{VersionId: projectScopeId(id), ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("clear version fork references %s: %w", id, err)
	}
	if err := q.DeleteVersionComponents(ctx, applicationsqlc.DeleteVersionComponentsParams{VersionId: id, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("delete version components %s: %w", id, err)
	}
	if err := q.DeleteVersion(ctx, applicationsqlc.DeleteVersionParams{Id: id, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("delete version %s: %w", id, err)
	}
	return nil
}

func (r Repository) VersionComponentsByVersion(ctx context.Context, projectId string, versionId string) ([]model.VersionComponent, error) {
	rows, err := r.q(ctx).VersionComponentsByVersion(ctx, applicationsqlc.VersionComponentsByVersionParams{VersionId: versionId, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return nil, fmt.Errorf("list version components %s: %w", versionId, err)
	}
	items := make([]model.VersionComponent, 0, len(rows))
	for _, row := range rows {
		component, err := r.componentFromRow(ctx, r.q(ctx), row.Id, row.VersionId, row.Name, row.Image, row.ArtifactId, row.ArtifactName, row.ArtifactImageRef, row.ArtifactLocalImageSha256, row.ArtifactSourceCommitSha, row.EntrypointJson, row.CommandJson, row.PullPolicy, row.RestartPolicy, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, component)
	}
	return items, nil
}

func (r Repository) VersionComponent(ctx context.Context, projectId string, id string) (model.VersionComponent, error) {
	row, err := r.q(ctx).VersionComponentById(ctx, applicationsqlc.VersionComponentByIdParams{Id: id, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component %s: %w", id, sqlcommon.TranslateError(err))
	}
	component, err := r.componentFromRow(ctx, r.q(ctx), row.Id, row.VersionId, row.Name, row.Image, row.ArtifactId, row.ArtifactName, row.ArtifactImageRef, row.ArtifactLocalImageSha256, row.ArtifactSourceCommitSha, row.EntrypointJson, row.CommandJson, row.PullPolicy, row.RestartPolicy, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return model.VersionComponent{}, err
	}
	return component, nil
}

func (r Repository) ReplaceVersionComponents(ctx context.Context, projectId string, versionId string, components []model.VersionComponent) error {
	return r.replaceVersionComponents(ctx, projectId, versionId, components)
}

func (r Repository) replaceVersionComponents(ctx context.Context, projectId string, versionId string, components []model.VersionComponent) error {
	q := r.q(ctx)
	if err := q.DeleteVersionComponents(ctx, applicationsqlc.DeleteVersionComponentsParams{VersionId: versionId, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("delete version components %s: %w", versionId, err)
	}
	now := time.Now().UTC()
	for _, c := range components {
		c.VersionId = versionId
		if err := insertVersionComponent(ctx, q, c, now); err != nil {
			return err
		}
	}
	if err := q.UpdateVersionComponentSummary(ctx, applicationsqlc.UpdateVersionComponentSummaryParams{
		ComponentSummary: model.VersionComponentSummary(components),
		UpdatedAt:        time.Now().UTC(),
		Id:               versionId,
		ProjectId:        projectScopeId(projectId),
	}); err != nil {
		return fmt.Errorf("update version component summary %s: %w", versionId, err)
	}
	return nil
}

func (r Repository) CreateVersionComponent(ctx context.Context, projectId string, component model.VersionComponent) error {
	q := r.q(ctx)
	if _, err := r.Version(ctx, projectId, component.VersionId); err != nil {
		return err
	}
	if err := insertVersionComponent(ctx, q, component, time.Now().UTC()); err != nil {
		return err
	}
	return r.updateComponentSummary(ctx, q, projectId, component.VersionId)
}

func (r Repository) UpdateVersionComponentBasic(ctx context.Context, projectId string, component model.VersionComponent, oldName string) error {
	q := r.q(ctx)
	now := time.Now().UTC()
	entrypointJSON, err := commandJSON(component.Entrypoint)
	if err != nil {
		return fmt.Errorf("encode component entrypoint: %w", err)
	}
	commandJSON, err := commandJSON(component.Command)
	if err != nil {
		return fmt.Errorf("encode component command: %w", err)
	}
	if err := q.UpdateVersionComponentBasic(ctx, applicationsqlc.UpdateVersionComponentBasicParams{
		Name:          component.Name,
		Image:         component.Image,
		PullPolicy:    component.PullPolicy,
		RestartPolicy: dbmodel.NullString(component.RestartPolicy),
		UpdatedAt:     now,
		Id:            component.Id,
		ProjectId:     projectScopeId(projectId),
	}); err != nil {
		return fmt.Errorf("update version component %s: %w", component.Id, err)
	}
	if err := q.UpdateVersionComponentCommand(ctx, applicationsqlc.UpdateVersionComponentCommandParams{
		CommandJson: commandJSON, UpdatedAt: now, Id: component.Id, ProjectId: projectScopeId(projectId),
	}); err != nil {
		return fmt.Errorf("update component command: %w", err)
	}
	if err := q.UpdateVersionComponentEntrypoint(ctx, applicationsqlc.UpdateVersionComponentEntrypointParams{
		EntrypointJson: entrypointJSON, UpdatedAt: now, Id: component.Id, ProjectId: projectScopeId(projectId),
	}); err != nil {
		return fmt.Errorf("update component entrypoint: %w", err)
	}
	if oldName != component.Name {
		if err := q.RenameVersionComponentDependencies(ctx, applicationsqlc.RenameVersionComponentDependenciesParams{
			NewName: component.Name, VersionId: component.VersionId, OldName: oldName, ProjectId: projectScopeId(projectId),
		}); err != nil {
			return fmt.Errorf("rename version component dependencies: %w", err)
		}
	}
	return r.updateComponentSummary(ctx, q, projectId, component.VersionId)
}

func (r Repository) UpdateVersionComponentRuntime(ctx context.Context, projectId string, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, projectId, component, deleteVersionComponentRuntimeConfig, insertVersionComponentRuntimeConfig)
}

func (r Repository) UpdateVersionComponentEndpoints(ctx context.Context, projectId string, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, projectId, component, deleteVersionComponentEndpointsConfig, insertVersionComponentEndpointsConfig)
}

func (r Repository) UpdateVersionComponentEnv(ctx context.Context, projectId string, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, projectId, component, deleteVersionComponentEnvConfig, insertVersionComponentEnvConfig)
}

func (r Repository) UpdateVersionComponentMounts(ctx context.Context, projectId string, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, projectId, component, deleteVersionComponentMountsConfig, insertVersionComponentMountsConfig)
}

func (r Repository) UpdateVersionComponentDependencies(ctx context.Context, projectId string, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, projectId, component, deleteVersionComponentDependenciesConfig, insertVersionComponentDependenciesConfig)
}

func (r Repository) UpdateVersionComponentAdvanced(ctx context.Context, projectId string, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, projectId, component, deleteVersionComponentAdvancedConfig, insertVersionComponentAdvancedConfig)
}

func (r Repository) UpdateVersionComponentDevices(ctx context.Context, projectId string, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, projectId, component, deleteVersionComponentDevicesConfig, insertVersionComponentDevicesConfig)
}

func (r Repository) updateVersionComponentConfig(ctx context.Context, projectId string, component model.VersionComponent, deleteConfig func(context.Context, *applicationsqlc.Queries, string) error, insertConfig func(context.Context, *applicationsqlc.Queries, model.VersionComponent) error) error {
	q := r.q(ctx)
	if err := q.TouchVersionComponent(ctx, applicationsqlc.TouchVersionComponentParams{UpdatedAt: time.Now().UTC(), Id: component.Id, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("touch version component %s: %w", component.Id, err)
	}
	if err := deleteConfig(ctx, q, component.Id); err != nil {
		return err
	}
	if err := insertConfig(ctx, q, component); err != nil {
		return err
	}
	return r.updateComponentSummary(ctx, q, projectId, component.VersionId)
}

func (r Repository) DeleteVersionComponent(ctx context.Context, projectId string, component model.VersionComponent) error {
	q := r.q(ctx)
	if err := q.DeleteVersionComponent(ctx, applicationsqlc.DeleteVersionComponentParams{Id: component.Id, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("delete version component %s: %w", component.Id, err)
	}
	return r.updateComponentSummary(ctx, q, projectId, component.VersionId)
}

func (r Repository) CreateVersionWithVersionComponents(ctx context.Context, projectId string, version model.Version, components []model.VersionComponent) error {
	if err := r.CreateVersion(ctx, projectId, version); err != nil {
		return err
	}
	if err := r.replaceVersionComponents(ctx, projectId, version.Id, components); err != nil {
		return err
	}
	return nil
}

func appFrom(id string, projectId sql.NullString, name, code, kind string, createdAt, updatedAt time.Time) model.Application {
	return model.Application{
		Id:        id,
		ProjectId: dbmodel.StringPtr(projectId),
		Name:      name,
		Code:      code,
		Kind:      kind,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func versionFrom(row applicationsqlc.Version) model.Version {
	return model.Version{
		Id:                   row.Id,
		ApplicationId:        row.ApplicationId,
		Label:                row.Label,
		Status:               row.Status,
		CreatedFromVersionId: dbmodel.StringPtr(row.CreatedFromVersionId),
		Note:                 dbmodel.StringPtr(row.Note),
		ComponentSummary:     row.ComponentSummary,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func (r Repository) componentFromRow(ctx context.Context, q *applicationsqlc.Queries, id string, versionId string, name string, image string, artifactId, artifactName, artifactImageRef, artifactLocalImageSha256, artifactSourceCommitSha sql.NullString, entrypointValue string, commandValue string, pullPolicy string, restartPolicy sql.NullString, createdAt time.Time, updatedAt time.Time) (model.VersionComponent, error) {
	entrypoint, err := commandFromJSON(entrypointValue)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("decode version component entrypoint %s: %w", id, err)
	}
	command, err := commandFromJSON(commandValue)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("decode version component command %s: %w", id, err)
	}
	component := model.VersionComponent{
		Id: id, VersionId: versionId, Name: name, Image: image, ArtifactId: dbmodel.StringPtr(artifactId),
		Entrypoint: entrypoint, Command: command,
		PullPolicy: pullPolicy, RestartPolicy: dbmodel.StringPtr(restartPolicy),
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
	env, err := q.VersionComponentEnvByComponent(ctx, component.Id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component env %s: %w", component.Id, err)
	}
	for _, item := range env {
		component.Env = append(component.Env, model.VersionComponentEnv{Key: item.EnvKey, Value: item.Value})
	}
	endpoints, err := q.VersionComponentEndpointsByComponent(ctx, component.Id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component endpoints %s: %w", component.Id, err)
	}
	for _, item := range endpoints {
		component.Endpoints = append(component.Endpoints, model.VersionComponentEndpoint{
			Protocol: item.Protocol, ContainerPort: int(item.ContainerPort), Mode: item.Mode,
			BindAddress: dbmodel.StringPtr(item.BindAddress), ListenPort: dbmodel.IntPtrFromNullInt64(item.ListenPort),
			Entrypoint: dbmodel.StringPtr(item.Entrypoint), PathPrefix: dbmodel.StringPtr(item.PathPrefix),
		})
	}
	mounts, err := q.VersionComponentMountsByComponent(ctx, component.Id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component mounts %s: %w", component.Id, err)
	}
	for _, item := range mounts {
		component.Mounts = append(component.Mounts, model.VersionComponentMount{
			SourceType: item.SourceType, Source: item.Source, Target: item.Target, ReadOnly: dbmodel.IntBool(item.ReadOnly),
			SourceIsHostPath: dbmodel.IntBool(item.SourceIsHostPath), Content: ptrValue(dbmodel.StringPtr(item.Content)),
			Mode: item.Mode, IgnoreIfExists: dbmodel.IntBool(item.IgnoreIfExists),
		})
	}
	dependencies, err := q.VersionComponentDependenciesByComponent(ctx, component.Id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component dependencies %s: %w", component.Id, err)
	}
	for _, item := range dependencies {
		component.Dependencies = append(component.Dependencies, model.VersionComponentDependency{Name: item.DependsOnName, Condition: item.Condition})
	}
	healthcheck, err := q.VersionComponentHealthcheckByComponent(ctx, component.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.VersionComponent{}, fmt.Errorf("load version component healthcheck %s: %w", component.Id, err)
	}
	if err == nil {
		component.Healthcheck = &model.VersionComponentHealthcheck{
			TestMode: healthcheck.TestMode.String, Test: healthcheck.Test, Interval: dbmodel.StringPtr(healthcheck.Interval), Timeout: dbmodel.StringPtr(healthcheck.Timeout),
			Retries: dbmodel.IntPtrFromNullInt64(healthcheck.Retries), StartPeriod: dbmodel.StringPtr(healthcheck.StartPeriod),
			StartInterval: dbmodel.StringPtr(healthcheck.StartInterval), Disabled: dbmodel.IntBool(healthcheck.Disabled),
		}
	}
	resource, err := q.VersionComponentResourceByComponent(ctx, component.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.VersionComponent{}, fmt.Errorf("load version component resource %s: %w", component.Id, err)
	}
	if err == nil {
		component.Resources = &model.VersionComponentResources{
			LimitCPUs: dbmodel.StringPtr(resource.LimitCpus), LimitMemory: dbmodel.StringPtr(resource.LimitMemory),
			ReservationCPUs: dbmodel.StringPtr(resource.ReservationCpus), ReservationMemory: dbmodel.StringPtr(resource.ReservationMemory),
		}
	}
	tmpfs, err := q.VersionComponentTmpfsByComponent(ctx, component.Id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component tmpfs %s: %w", component.Id, err)
	}
	for _, item := range tmpfs {
		component.Tmpfs = append(component.Tmpfs, model.VersionComponentTmpfs{Target: item.Target, SizeBytes: item.SizeBytes, Mode: item.Mode})
	}
	ulimits, err := q.VersionComponentUlimitsByComponent(ctx, component.Id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component ulimits %s: %w", component.Id, err)
	}
	for _, item := range ulimits {
		component.Ulimits = append(component.Ulimits, model.VersionComponentUlimit{Name: item.Name, Soft: item.Soft, Hard: item.Hard})
	}
	devices, err := q.VersionComponentDevicesByComponent(ctx, component.Id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component devices %s: %w", component.Id, err)
	}
	for _, item := range devices {
		var capabilities []string
		if err := json.Unmarshal([]byte(item.CapabilitiesJson), &capabilities); err != nil || capabilities == nil {
			if err == nil {
				err = fmt.Errorf("must be a JSON array")
			}
			return model.VersionComponent{}, fmt.Errorf("decode version component device %s: %w", component.Id, err)
		}
		component.Devices = append(component.Devices, model.VersionComponentDeviceRequest{
			Driver: item.Driver, Count: item.DeviceCount, Capabilities: capabilities,
		})
	}
	if artifactId.Valid {
		component.Artifact = &model.VersionComponentArtifact{
			ArtifactId: artifactId.String, ArtifactName: artifactName.String, ImageRef: artifactImageRef.String,
			LocalImageSha256: artifactLocalImageSha256.String, SourceCommitSha: artifactSourceCommitSha.String,
		}
	}
	return component, nil
}

func insertVersionComponent(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent, fallback time.Time) error {
	entrypointJSON, err := commandJSON(component.Entrypoint)
	if err != nil {
		return fmt.Errorf("encode version component entrypoint %s: %w", component.Name, err)
	}
	commandJSON, err := commandJSON(component.Command)
	if err != nil {
		return fmt.Errorf("encode version component command %s: %w", component.Name, err)
	}
	createdAt, updatedAt := component.CreatedAt, component.UpdatedAt
	if createdAt.IsZero() {
		createdAt = fallback
	}
	if updatedAt.IsZero() {
		updatedAt = fallback
	}
	artifactId := dbmodel.NullString(component.ArtifactId)
	artifactName, artifactImageRef, artifactLocalImageSha256, artifactSourceCommitSha := sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{}
	if artifact := component.Artifact; artifact != nil {
		artifactId = sql.NullString{String: artifact.ArtifactId, Valid: artifact.ArtifactId != ""}
		artifactName = sql.NullString{String: artifact.ArtifactName, Valid: artifact.ArtifactName != ""}
		artifactImageRef = sql.NullString{String: artifact.ImageRef, Valid: artifact.ImageRef != ""}
		artifactLocalImageSha256 = sql.NullString{String: artifact.LocalImageSha256, Valid: artifact.LocalImageSha256 != ""}
		artifactSourceCommitSha = sql.NullString{String: artifact.SourceCommitSha, Valid: artifact.SourceCommitSha != ""}
	}
	if err := q.InsertVersionComponent(ctx, applicationsqlc.InsertVersionComponentParams{
		Id: component.Id, VersionId: component.VersionId, Name: component.Name, Image: component.Image, ArtifactId: artifactId,
		ArtifactName: artifactName, ArtifactImageRef: artifactImageRef, ArtifactLocalImageSha256: artifactLocalImageSha256, ArtifactSourceCommitSha: artifactSourceCommitSha, EntrypointJson: entrypointJSON, CommandJson: commandJSON,
		PullPolicy: component.PullPolicy, RestartPolicy: dbmodel.NullString(component.RestartPolicy),
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}); err != nil {
		return fmt.Errorf("insert version component %s: %w", component.Name, err)
	}
	return insertVersionComponentConfig(ctx, q, component)
}

func insertVersionComponentConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for _, insertConfig := range []func(context.Context, *applicationsqlc.Queries, model.VersionComponent) error{
		insertVersionComponentRuntimeConfig,
		insertVersionComponentConnectivityConfig,
		insertVersionComponentAdvancedConfig,
		insertVersionComponentDevicesConfig,
	} {
		if err := insertConfig(ctx, q, component); err != nil {
			return err
		}
	}
	return nil
}

func insertVersionComponentRuntimeConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	if component.Healthcheck != nil {
		if err := q.InsertVersionComponentHealthcheck(ctx, applicationsqlc.InsertVersionComponentHealthcheckParams{
			ComponentId: component.Id, TestMode: optionalText(component.Healthcheck.TestMode), Test: component.Healthcheck.Test, Interval: dbmodel.NullString(component.Healthcheck.Interval),
			Timeout: dbmodel.NullString(component.Healthcheck.Timeout), Retries: dbmodel.NullInt64FromIntPtr(component.Healthcheck.Retries),
			StartPeriod: dbmodel.NullString(component.Healthcheck.StartPeriod), StartInterval: dbmodel.NullString(component.Healthcheck.StartInterval),
			Disabled: dbmodel.BoolInt(component.Healthcheck.Disabled),
		}); err != nil {
			return fmt.Errorf("insert component healthcheck: %w", err)
		}
	}
	return nil
}

func insertVersionComponentConnectivityConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for _, insertConfig := range []func(context.Context, *applicationsqlc.Queries, model.VersionComponent) error{
		insertVersionComponentEnvConfig,
		insertVersionComponentEndpointsConfig,
		insertVersionComponentMountsConfig,
		insertVersionComponentDependenciesConfig,
	} {
		if err := insertConfig(ctx, q, component); err != nil {
			return err
		}
	}
	return nil
}

func insertVersionComponentEnvConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for position, item := range component.Env {
		if err := q.InsertVersionComponentEnv(ctx, applicationsqlc.InsertVersionComponentEnvParams{ComponentId: component.Id, EnvKey: item.Key, Value: item.Value, Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component env: %w", err)
		}
	}
	return nil
}

func insertVersionComponentEndpointsConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for position, item := range component.Endpoints {
		if err := q.InsertVersionComponentEndpoint(ctx, applicationsqlc.InsertVersionComponentEndpointParams{ComponentId: component.Id, Protocol: item.Protocol, ContainerPort: int64(item.ContainerPort), Mode: item.Mode, BindAddress: dbmodel.NullString(item.BindAddress), ListenPort: dbmodel.NullInt64FromIntPtr(item.ListenPort), Entrypoint: dbmodel.NullString(item.Entrypoint), PathPrefix: dbmodel.NullString(item.PathPrefix), Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component endpoint: %w", err)
		}
	}
	return nil
}

func insertVersionComponentMountsConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for position, item := range component.Mounts {
		if err := q.InsertVersionComponentMount(ctx, applicationsqlc.InsertVersionComponentMountParams{
			ComponentId: component.Id, SourceType: item.SourceType, Source: item.Source, Target: item.Target, ReadOnly: dbmodel.BoolInt(item.ReadOnly),
			SourceIsHostPath: dbmodel.BoolInt(item.SourceIsHostPath), Content: optionalText(item.Content),
			Mode: item.Mode, IgnoreIfExists: dbmodel.BoolInt(item.IgnoreIfExists), Position: int64(position),
		}); err != nil {
			return fmt.Errorf("insert component mount: %w", err)
		}
	}
	return nil
}

func insertVersionComponentDependenciesConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for position, item := range component.Dependencies {
		if err := q.InsertVersionComponentDependency(ctx, applicationsqlc.InsertVersionComponentDependencyParams{ComponentId: component.Id, DependsOnName: item.Name, Condition: item.Condition, Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component dependency: %w", err)
		}
	}
	return nil
}

func insertVersionComponentAdvancedConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	if component.Resources != nil {
		if err := q.InsertVersionComponentResource(ctx, applicationsqlc.InsertVersionComponentResourceParams{
			ComponentId: component.Id, LimitCpus: dbmodel.NullString(component.Resources.LimitCPUs), LimitMemory: dbmodel.NullString(component.Resources.LimitMemory),
			ReservationCpus: dbmodel.NullString(component.Resources.ReservationCPUs), ReservationMemory: dbmodel.NullString(component.Resources.ReservationMemory),
		}); err != nil {
			return fmt.Errorf("insert component resources: %w", err)
		}
	}
	for position, item := range component.Tmpfs {
		if err := q.InsertVersionComponentTmpfs(ctx, applicationsqlc.InsertVersionComponentTmpfsParams{ComponentId: component.Id, Target: item.Target, SizeBytes: item.SizeBytes, Mode: item.Mode, Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component tmpfs: %w", err)
		}
	}
	for position, item := range component.Ulimits {
		if err := q.InsertVersionComponentUlimit(ctx, applicationsqlc.InsertVersionComponentUlimitParams{ComponentId: component.Id, Name: item.Name, Soft: item.Soft, Hard: item.Hard, Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component ulimit: %w", err)
		}
	}
	return nil
}

func insertVersionComponentDevicesConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for position, item := range component.Devices {
		capabilitiesJSON, err := json.Marshal(item.Capabilities)
		if err != nil {
			return fmt.Errorf("encode component device capabilities: %w", err)
		}
		if err := q.InsertVersionComponentDevice(ctx, applicationsqlc.InsertVersionComponentDeviceParams{
			ComponentId: component.Id, Driver: item.Driver, DeviceCount: item.Count,
			CapabilitiesJson: string(capabilitiesJSON), Position: int64(position),
		}); err != nil {
			return fmt.Errorf("insert component device: %w", err)
		}
	}
	return nil
}

func deleteVersionComponentRuntimeConfig(ctx context.Context, q *applicationsqlc.Queries, componentId string) error {
	return deleteVersionComponentConfigRows(ctx, componentId,
		q.DeleteVersionComponentHealthcheck,
	)
}

func commandFromJSON(value string) ([]string, error) {
	var command []string
	if err := json.Unmarshal([]byte(value), &command); err != nil {
		return nil, err
	}
	if command == nil {
		return nil, fmt.Errorf("must be a JSON array")
	}
	return command, nil
}

func commandJSON(command []string) (string, error) {
	if command == nil {
		command = []string{}
	}
	encoded, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func deleteVersionComponentEndpointsConfig(ctx context.Context, q *applicationsqlc.Queries, componentId string) error {
	return deleteVersionComponentConfigRows(ctx, componentId, q.DeleteVersionComponentEndpoints)
}

func deleteVersionComponentEnvConfig(ctx context.Context, q *applicationsqlc.Queries, componentId string) error {
	return deleteVersionComponentConfigRows(ctx, componentId, q.DeleteVersionComponentEnv)
}

func deleteVersionComponentMountsConfig(ctx context.Context, q *applicationsqlc.Queries, componentId string) error {
	return deleteVersionComponentConfigRows(ctx, componentId, q.DeleteVersionComponentMounts)
}

func deleteVersionComponentDependenciesConfig(ctx context.Context, q *applicationsqlc.Queries, componentId string) error {
	return deleteVersionComponentConfigRows(ctx, componentId, q.DeleteVersionComponentDependencies)
}

func deleteVersionComponentAdvancedConfig(ctx context.Context, q *applicationsqlc.Queries, componentId string) error {
	return deleteVersionComponentConfigRows(ctx, componentId,
		q.DeleteVersionComponentResource,
		q.DeleteVersionComponentTmpfs,
		q.DeleteVersionComponentUlimits,
	)
}

func deleteVersionComponentDevicesConfig(ctx context.Context, q *applicationsqlc.Queries, componentId string) error {
	return deleteVersionComponentConfigRows(ctx, componentId, q.DeleteVersionComponentDevices)
}

func deleteVersionComponentConfigRows(ctx context.Context, componentId string, deletes ...func(context.Context, string) error) error {
	for _, deleteConfig := range deletes {
		if err := deleteConfig(ctx, componentId); err != nil {
			return fmt.Errorf("delete component config %s: %w", componentId, err)
		}
	}
	return nil
}

func (r Repository) updateComponentSummary(ctx context.Context, q *applicationsqlc.Queries, projectId string, versionId string) error {
	rows, err := q.VersionComponentsByVersion(ctx, applicationsqlc.VersionComponentsByVersionParams{VersionId: versionId, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return fmt.Errorf("list version components for summary %s: %w", versionId, err)
	}
	components := make([]model.VersionComponent, 0, len(rows))
	for _, row := range rows {
		components = append(components, model.VersionComponent{Name: row.Name, Image: row.Image})
	}
	if err := q.UpdateVersionComponentSummary(ctx, applicationsqlc.UpdateVersionComponentSummaryParams{
		ComponentSummary: model.VersionComponentSummary(components), UpdatedAt: time.Now().UTC(), Id: versionId, ProjectId: projectScopeId(projectId),
	}); err != nil {
		return fmt.Errorf("update version component summary %s: %w", versionId, err)
	}
	return nil
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalText(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
