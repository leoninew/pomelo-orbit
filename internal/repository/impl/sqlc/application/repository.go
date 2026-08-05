package applicationrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	applicationsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/application"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
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

func optionalProjectId(projectId *string) interface{} {
	if projectId == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*projectId)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func (r Repository) ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	kindFilter := strings.TrimSpace(kind)
	q := r.q(ctx)
	total, err := q.CountApplications(ctx, applicationsqlc.CountApplicationsParams{
		ProjectID:   optionalProjectId(projectId),
		Search:      raw,
		NamePattern: pattern,
		KindFilter:  kindFilter,
		Kind:        kindFilter,
	})
	if err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("count applications: %w", err)
	}
	rows, err := q.ListApplications(ctx, applicationsqlc.ListApplicationsParams{
		ProjectID:   optionalProjectId(projectId),
		Search:      raw,
		NamePattern: pattern,
		KindFilter:  kindFilter,
		Kind:        kindFilter,
		Limit:       int64(perPage),
		Offset:      int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("list applications: %w", err)
	}
	items := make([]model.Application, 0, len(rows))
	for _, row := range rows {
		items = append(items, appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.Application]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Application(ctx context.Context, id string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByID(ctx, id)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application %s: %w", id, sqlcommon.TranslateError(err))
	}
	return appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByName(ctx context.Context, name string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByName(ctx, name)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByCode(ctx context.Context, code string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByCode(ctx, code)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.CreatedAt, row.UpdatedAt), nil
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
		ID:        app.Id,
		ProjectID: dbmodel.NullString(app.ProjectId),
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

func (r Repository) UpdateApplication(ctx context.Context, app model.Application) error {
	err := r.q(ctx).UpdateApplication(ctx, applicationsqlc.UpdateApplicationParams{
		Name:      app.Name,
		Code:      app.Code,
		UpdatedAt: time.Now().UTC(),
		ID:        app.Id,
	})
	if err != nil {
		return fmt.Errorf("update application %s: %w", app.Id, err)
	}
	return nil
}

func (r Repository) DeleteApplication(ctx context.Context, id string) error {
	q := r.q(ctx)
	versionIds, err := q.VersionIdsByApplication(ctx, id)
	if err != nil {
		return fmt.Errorf("list application version ids %s: %w", id, err)
	}
	for _, versionId := range versionIds {
		refs, err := q.CountVersionRuntimeRefs(ctx, versionId)
		if err != nil {
			return fmt.Errorf("count version references %s: %w", versionId, err)
		}
		if asInt(refs) > 0 {
			return fmt.Errorf("application %s contains referenced version %s: %w", id, versionId, repository.ErrReferenced)
		}
	}
	if err := q.DetachDeploymentServiceRefsByApplication(ctx, id); err != nil {
		return fmt.Errorf("detach deployment service refs for application %s: %w", id, err)
	}
	if err := q.DetachDeploymentVersionRefsByApplication(ctx, id); err != nil {
		return fmt.Errorf("detach deployment version refs for application %s: %w", id, err)
	}
	if err := q.DeleteServicesByApplication(ctx, id); err != nil {
		return fmt.Errorf("delete application service %s: %w", id, err)
	}
	if err := q.DeleteVersionsByApplication(ctx, id); err != nil {
		return fmt.Errorf("delete application versions %s: %w", id, err)
	}
	if err := q.DeleteApplication(ctx, id); err != nil {
		return fmt.Errorf("delete application %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListVersions(ctx context.Context, applicationId string) ([]model.Version, error) {
	rows, err := r.q(ctx).ListVersions(ctx, applicationId)
	if err != nil {
		return nil, fmt.Errorf("list versions %s: %w", applicationId, err)
	}
	items := make([]model.Version, 0, len(rows))
	for _, row := range rows {
		items = append(items, versionFrom(row))
	}
	return items, nil
}

func (r Repository) LatestVersionByApplication(ctx context.Context, applicationId string) (model.Version, error) {
	row, err := r.q(ctx).LatestVersionByApplication(ctx, applicationId)
	if err != nil {
		return model.Version{}, fmt.Errorf("load latest version for application %s: %w", applicationId, sqlcommon.TranslateError(err))
	}
	return versionFrom(row), nil
}

func (r Repository) ListVersionsPage(ctx context.Context, applicationId string, page int, perPage int, search string) (repository.Page[model.Version], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	notePattern := sql.NullString{String: pattern, Valid: true}
	q := r.q(ctx)
	total, err := q.CountVersions(ctx, applicationsqlc.CountVersionsParams{
		ApplicationID: applicationId,
		Column2:       raw,
		Label:         pattern,
		Note:          notePattern,
	})
	if err != nil {
		return repository.Page[model.Version]{}, fmt.Errorf("count versions: %w", err)
	}
	rows, err := q.ListVersionsPage(ctx, applicationsqlc.ListVersionsPageParams{
		ApplicationID: applicationId,
		Column2:       raw,
		Label:         pattern,
		Note:          notePattern,
		Limit:         int64(perPage),
		Offset:        int64((page - 1) * perPage),
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

func (r Repository) Version(ctx context.Context, id string) (model.Version, error) {
	row, err := r.q(ctx).VersionByID(ctx, id)
	if err != nil {
		return model.Version{}, fmt.Errorf("load version %s: %w", id, sqlcommon.TranslateError(err))
	}
	return versionFrom(row), nil
}

func (r Repository) CreateVersion(ctx context.Context, version model.Version) error {
	now := time.Now().UTC()
	createdAt, updatedAt := version.CreatedAt, version.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	err := r.q(ctx).CreateVersion(ctx, applicationsqlc.CreateVersionParams{
		ID:                   version.Id,
		ApplicationID:        version.ApplicationId,
		Label:                version.Label,
		Status:               version.Status,
		CreatedFromVersionID: dbmodel.NullString(version.CreatedFromVersionId),
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

func (r Repository) UpdateVersion(ctx context.Context, version model.Version) error {
	err := r.q(ctx).UpdateVersion(ctx, applicationsqlc.UpdateVersionParams{
		Label:            version.Label,
		Status:           version.Status,
		Note:             dbmodel.NullString(version.Note),
		ComponentSummary: version.ComponentSummary,
		UpdatedAt:        time.Now().UTC(),
		ID:               version.Id,
	})
	if err != nil {
		return fmt.Errorf("update version %s: %w", version.Id, err)
	}
	return nil
}

func (r Repository) DeleteVersion(ctx context.Context, id string) error {
	q := r.q(ctx)
	if err := q.DeleteVersionComponents(ctx, id); err != nil {
		return fmt.Errorf("delete version components %s: %w", id, err)
	}
	if err := q.DeleteVersion(ctx, id); err != nil {
		return fmt.Errorf("delete version %s: %w", id, err)
	}
	return nil
}

func (r Repository) CountVersionRuntimeRefs(ctx context.Context, versionId string) (int, error) {
	raw, err := r.q(ctx).CountVersionRuntimeRefs(ctx, versionId)
	if err != nil {
		return 0, fmt.Errorf("count version runtime refs %s: %w", versionId, err)
	}
	return asInt(raw), nil
}

func (r Repository) VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error) {
	rows, err := r.q(ctx).VersionComponentsByVersion(ctx, versionId)
	if err != nil {
		return nil, fmt.Errorf("list version components %s: %w", versionId, err)
	}
	items := make([]model.VersionComponent, 0, len(rows))
	for _, row := range rows {
		component, err := r.componentFromRow(ctx, r.q(ctx), row.ID, row.VersionID, row.Name, row.Image, row.ArtifactID, row.CommandJson, row.PullPolicy, row.RestartPolicy, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, component)
	}
	return items, nil
}

func (r Repository) VersionComponent(ctx context.Context, id string) (model.VersionComponent, error) {
	row, err := r.q(ctx).VersionComponentByID(ctx, id)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("load version component %s: %w", id, sqlcommon.TranslateError(err))
	}
	component, err := r.componentFromRow(ctx, r.q(ctx), row.ID, row.VersionID, row.Name, row.Image, row.ArtifactID, row.CommandJson, row.PullPolicy, row.RestartPolicy, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return model.VersionComponent{}, err
	}
	return component, nil
}

func (r Repository) SetVersionComponentArtifact(ctx context.Context, componentId string, artifactId string) error {
	if err := r.q(ctx).SetVersionComponentArtifact(ctx, applicationsqlc.SetVersionComponentArtifactParams{
		ArtifactID: sql.NullString{String: artifactId, Valid: true}, ComponentID: componentId,
	}); err != nil {
		return fmt.Errorf("set version component artifact %s: %w", componentId, err)
	}
	return nil
}

func (r Repository) ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		return r.replaceVersionComponents(txCtx, versionId, components)
	})
}

func (r Repository) replaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error {
	q := r.q(ctx)
	if err := q.DeleteVersionComponents(ctx, versionId); err != nil {
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
		ID:               versionId,
	}); err != nil {
		return fmt.Errorf("update version component summary %s: %w", versionId, err)
	}
	return nil
}

func (r Repository) CreateVersionComponent(ctx context.Context, component model.VersionComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := insertVersionComponent(txCtx, q, component, time.Now().UTC()); err != nil {
			return err
		}
		return r.updateComponentSummary(txCtx, q, component.VersionId)
	})
}

func (r Repository) UpdateVersionComponentBasic(ctx context.Context, component model.VersionComponent, oldName string) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		now := time.Now().UTC()
		commandJSON, err := commandJSON(component.Command)
		if err != nil {
			return fmt.Errorf("encode component command: %w", err)
		}
		if err := q.UpdateVersionComponentBasic(txCtx, applicationsqlc.UpdateVersionComponentBasicParams{
			Name:          component.Name,
			Image:         component.Image,
			PullPolicy:    component.PullPolicy,
			RestartPolicy: dbmodel.NullString(component.RestartPolicy),
			UpdatedAt:     now,
			ID:            component.Id,
		}); err != nil {
			return fmt.Errorf("update version component %s: %w", component.Id, err)
		}
		if err := q.UpdateVersionComponentCommand(txCtx, applicationsqlc.UpdateVersionComponentCommandParams{
			CommandJson: commandJSON, UpdatedAt: now, ID: component.Id,
		}); err != nil {
			return fmt.Errorf("update component command: %w", err)
		}
		if oldName != component.Name {
			if err := q.RenameVersionComponentDependencies(txCtx, applicationsqlc.RenameVersionComponentDependenciesParams{
				NewName: component.Name, VersionID: component.VersionId, OldName: oldName,
			}); err != nil {
				return fmt.Errorf("rename version component dependencies: %w", err)
			}
		}
		return r.updateComponentSummary(txCtx, q, component.VersionId)
	})
}

func (r Repository) UpdateVersionComponentRuntime(ctx context.Context, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, component, deleteVersionComponentRuntimeConfig, insertVersionComponentRuntimeConfig)
}

func (r Repository) UpdateVersionComponentEndpoints(ctx context.Context, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, component, deleteVersionComponentEndpointsConfig, insertVersionComponentEndpointsConfig)
}

func (r Repository) UpdateVersionComponentEnv(ctx context.Context, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, component, deleteVersionComponentEnvConfig, insertVersionComponentEnvConfig)
}

func (r Repository) UpdateVersionComponentMounts(ctx context.Context, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, component, deleteVersionComponentMountsConfig, insertVersionComponentMountsConfig)
}

func (r Repository) UpdateVersionComponentDependencies(ctx context.Context, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, component, deleteVersionComponentDependenciesConfig, insertVersionComponentDependenciesConfig)
}

func (r Repository) UpdateVersionComponentAdvanced(ctx context.Context, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, component, deleteVersionComponentAdvancedConfig, insertVersionComponentAdvancedConfig)
}

func (r Repository) UpdateVersionComponentDevices(ctx context.Context, component model.VersionComponent) error {
	return r.updateVersionComponentConfig(ctx, component, deleteVersionComponentDevicesConfig, insertVersionComponentDevicesConfig)
}

func (r Repository) updateVersionComponentConfig(ctx context.Context, component model.VersionComponent, deleteConfig func(context.Context, *applicationsqlc.Queries, string) error, insertConfig func(context.Context, *applicationsqlc.Queries, model.VersionComponent) error) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.TouchVersionComponent(txCtx, applicationsqlc.TouchVersionComponentParams{UpdatedAt: time.Now().UTC(), ID: component.Id}); err != nil {
			return fmt.Errorf("touch version component %s: %w", component.Id, err)
		}
		if err := deleteConfig(txCtx, q, component.Id); err != nil {
			return err
		}
		if err := insertConfig(txCtx, q, component); err != nil {
			return err
		}
		return r.updateComponentSummary(txCtx, q, component.VersionId)
	})
}

func (r Repository) DeleteVersionComponent(ctx context.Context, component model.VersionComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.DeleteVersionComponent(txCtx, component.Id); err != nil {
			return fmt.Errorf("delete version component %s: %w", component.Id, err)
		}
		return r.updateComponentSummary(txCtx, q, component.VersionId)
	})
}

func (r Repository) CreateVersionWithVersionComponents(ctx context.Context, version model.Version, components []model.VersionComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.CreateVersion(txCtx, version); err != nil {
			return err
		}
		if err := r.replaceVersionComponents(txCtx, version.Id, components); err != nil {
			return err
		}
		return nil
	})
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
		Id:                   row.ID,
		ApplicationId:        row.ApplicationID,
		Label:                row.Label,
		Status:               row.Status,
		CreatedFromVersionId: dbmodel.StringPtr(row.CreatedFromVersionID),
		Note:                 dbmodel.StringPtr(row.Note),
		ComponentSummary:     row.ComponentSummary,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func (r Repository) componentFromRow(ctx context.Context, q *applicationsqlc.Queries, id string, versionId string, name string, image string, artifactId sql.NullString, commandValue string, pullPolicy string, restartPolicy sql.NullString, createdAt time.Time, updatedAt time.Time) (model.VersionComponent, error) {
	command, err := commandFromJSON(commandValue)
	if err != nil {
		return model.VersionComponent{}, fmt.Errorf("decode version component command %s: %w", id, err)
	}
	component := model.VersionComponent{
		Id: id, VersionId: versionId, Name: name, Image: image, ArtifactId: dbmodel.StringPtr(artifactId),
		Command:    command,
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
			Name: item.Name, Protocol: item.Protocol, ContainerPort: int(item.ContainerPort), Mode: item.Mode,
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
	artifact, err := q.VersionComponentArtifact(ctx, component.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.VersionComponent{}, fmt.Errorf("load version component artifact %s: %w", component.Id, err)
	}
	if err == nil {
		component.Artifact = &model.VersionComponentArtifact{
			ArtifactId: artifact.ID, ImageRef: artifact.ImageRef.String,
			LocalImageSha256: artifact.LocalImageSha256.String, SourceCommitSha: artifact.SourceCommitSha.String,
		}
	}
	return component, nil
}

func insertVersionComponent(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent, fallback time.Time) error {
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
	if err := q.InsertVersionComponent(ctx, applicationsqlc.InsertVersionComponentParams{
		ID: component.Id, VersionID: component.VersionId, Name: component.Name, Image: component.Image, ArtifactID: dbmodel.NullString(component.ArtifactId),
		CommandJson: commandJSON,
		PullPolicy:  component.PullPolicy, RestartPolicy: dbmodel.NullString(component.RestartPolicy),
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
			ComponentID: component.Id, TestMode: optionalText(component.Healthcheck.TestMode), Test: component.Healthcheck.Test, Interval: dbmodel.NullString(component.Healthcheck.Interval),
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
		if err := q.InsertVersionComponentEnv(ctx, applicationsqlc.InsertVersionComponentEnvParams{ComponentID: component.Id, EnvKey: item.Key, Value: item.Value, Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component env: %w", err)
		}
	}
	return nil
}

func insertVersionComponentEndpointsConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for position, item := range component.Endpoints {
		if err := q.InsertVersionComponentEndpoint(ctx, applicationsqlc.InsertVersionComponentEndpointParams{ComponentID: component.Id, Name: item.Name, Protocol: item.Protocol, ContainerPort: int64(item.ContainerPort), Mode: item.Mode, BindAddress: dbmodel.NullString(item.BindAddress), ListenPort: dbmodel.NullInt64FromIntPtr(item.ListenPort), Entrypoint: dbmodel.NullString(item.Entrypoint), PathPrefix: dbmodel.NullString(item.PathPrefix), Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component endpoint: %w", err)
		}
	}
	return nil
}

func insertVersionComponentMountsConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	for position, item := range component.Mounts {
		if err := q.InsertVersionComponentMount(ctx, applicationsqlc.InsertVersionComponentMountParams{
			ComponentID: component.Id, SourceType: item.SourceType, Source: item.Source, Target: item.Target, ReadOnly: dbmodel.BoolInt(item.ReadOnly),
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
		if err := q.InsertVersionComponentDependency(ctx, applicationsqlc.InsertVersionComponentDependencyParams{ComponentID: component.Id, DependsOnName: item.Name, Condition: item.Condition, Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component dependency: %w", err)
		}
	}
	return nil
}

func insertVersionComponentAdvancedConfig(ctx context.Context, q *applicationsqlc.Queries, component model.VersionComponent) error {
	if component.Resources != nil {
		if err := q.InsertVersionComponentResource(ctx, applicationsqlc.InsertVersionComponentResourceParams{
			ComponentID: component.Id, LimitCpus: dbmodel.NullString(component.Resources.LimitCPUs), LimitMemory: dbmodel.NullString(component.Resources.LimitMemory),
			ReservationCpus: dbmodel.NullString(component.Resources.ReservationCPUs), ReservationMemory: dbmodel.NullString(component.Resources.ReservationMemory),
		}); err != nil {
			return fmt.Errorf("insert component resources: %w", err)
		}
	}
	for position, item := range component.Tmpfs {
		if err := q.InsertVersionComponentTmpfs(ctx, applicationsqlc.InsertVersionComponentTmpfsParams{ComponentID: component.Id, Target: item.Target, SizeBytes: item.SizeBytes, Mode: item.Mode, Position: int64(position)}); err != nil {
			return fmt.Errorf("insert component tmpfs: %w", err)
		}
	}
	for position, item := range component.Ulimits {
		if err := q.InsertVersionComponentUlimit(ctx, applicationsqlc.InsertVersionComponentUlimitParams{ComponentID: component.Id, Name: item.Name, Soft: item.Soft, Hard: item.Hard, Position: int64(position)}); err != nil {
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
			ComponentID: component.Id, Driver: item.Driver, DeviceCount: item.Count,
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

func (r Repository) updateComponentSummary(ctx context.Context, q *applicationsqlc.Queries, versionId string) error {
	rows, err := q.VersionComponentsByVersion(ctx, versionId)
	if err != nil {
		return fmt.Errorf("list version components for summary %s: %w", versionId, err)
	}
	components := make([]model.VersionComponent, 0, len(rows))
	for _, row := range rows {
		components = append(components, model.VersionComponent{Name: row.Name, Image: row.Image})
	}
	if err := q.UpdateVersionComponentSummary(ctx, applicationsqlc.UpdateVersionComponentSummaryParams{
		ComponentSummary: model.VersionComponentSummary(components), UpdatedAt: time.Now().UTC(), ID: versionId,
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

func asInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int32:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	case []byte:
		var parsed int64
		if _, err := fmt.Sscan(string(n), &parsed); err == nil {
			return int(parsed)
		}
	case string:
		var parsed int64
		if _, err := fmt.Sscan(n, &parsed); err == nil {
			return int(parsed)
		}
	}
	return 0
}
