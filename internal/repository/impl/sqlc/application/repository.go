package applicationrepo

import (
	"context"
	"database/sql"
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
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *applicationsqlc.Queries {
		return applicationsqlc.New(dbtx)
	})
}

func optionalProjectID(projectId *string) interface{} {
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
		ProjectID:   optionalProjectID(projectId),
		Search:      raw,
		NamePattern: pattern,
		KindFilter:  kindFilter,
		Kind:        kindFilter,
	})
	if err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("count applications: %w", err)
	}
	rows, err := q.ListApplications(ctx, applicationsqlc.ListApplicationsParams{
		ProjectID:   optionalProjectID(projectId),
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
		items = append(items, appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.ImagePullPolicy, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.Application]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Application(ctx context.Context, id string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByID(ctx, id)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application %s: %w", id, sqlcommon.TranslateError(err))
	}
	return appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.ImagePullPolicy, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByName(ctx context.Context, name string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByName(ctx, name)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.ImagePullPolicy, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) ApplicationByCode(ctx context.Context, code string) (model.Application, error) {
	row, err := r.q(ctx).ApplicationByCode(ctx, code)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return appFrom(row.ID, row.ProjectID, row.Name, row.Code, row.Kind, row.ImagePullPolicy, row.CreatedAt, row.UpdatedAt), nil
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
		ID:              app.Id,
		ProjectID:       dbmodel.NullString(app.ProjectId),
		Name:            app.Name,
		Code:            app.Code,
		Kind:            app.Kind,
		ImagePullPolicy: app.ImagePullPolicy,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	})
	if err != nil {
		return fmt.Errorf("create application %s: %w", app.Name, err)
	}
	return nil
}

func (r Repository) UpdateApplication(ctx context.Context, app model.Application) error {
	err := r.q(ctx).UpdateApplication(ctx, applicationsqlc.UpdateApplicationParams{
		Name:            app.Name,
		Code:            app.Code,
		ImagePullPolicy: app.ImagePullPolicy,
		UpdatedAt:       time.Now().UTC(),
		ID:              app.Id,
	})
	if err != nil {
		return fmt.Errorf("update application %s: %w", app.Id, err)
	}
	return nil
}

func (r Repository) DeleteApplication(ctx context.Context, id string) error {
	q := r.q(ctx)
	if err := q.DetachDeploymentServiceRefsByApplication(ctx, id); err != nil {
		return fmt.Errorf("detach deployment service refs for application %s: %w", id, err)
	}
	if err := q.DetachDeploymentVersionRefsByApplication(ctx, id); err != nil {
		return fmt.Errorf("detach deployment version refs for application %s: %w", id, err)
	}
	if err := q.DeleteServicesByApplication(ctx, id); err != nil {
		return fmt.Errorf("delete application service %s: %w", id, err)
	}
	if err := q.ClearVersionForkRefsByApplication(ctx, id); err != nil {
		return fmt.Errorf("detach version fork refs for application %s: %w", id, err)
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
		EnvJson:              dbmodel.NullString(version.EnvJSON),
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
		EnvJson:          dbmodel.NullString(version.EnvJSON),
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
	if err := q.ClearVersionForkRefs(ctx, sql.NullString{String: id, Valid: true}); err != nil {
		return fmt.Errorf("clear version fork refs %s: %w", id, err)
	}
	if err := q.DeleteVersionComponents(ctx, id); err != nil {
		return fmt.Errorf("delete version components %s: %w", id, err)
	}
	if err := q.DeleteVersionExposes(ctx, id); err != nil {
		return fmt.Errorf("delete version exposes %s: %w", id, err)
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
		items = append(items, componentFrom(row))
	}
	return items, nil
}

func (r Repository) VersionExposesByVersion(ctx context.Context, versionId string) ([]model.VersionExpose, error) {
	rows, err := r.q(ctx).VersionExposesByVersion(ctx, versionId)
	if err != nil {
		return nil, fmt.Errorf("list version exposes %s: %w", versionId, err)
	}
	items := make([]model.VersionExpose, 0, len(rows))
	for _, row := range rows {
		items = append(items, exposeFrom(row))
	}
	return items, nil
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
		createdAt, updatedAt := c.CreatedAt, c.UpdatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		if updatedAt.IsZero() {
			updatedAt = now
		}
		if err := q.InsertVersionComponent(ctx, applicationsqlc.InsertVersionComponentParams{
			ID:              c.Id,
			VersionID:       versionId,
			Name:            c.Name,
			Image:           c.Image,
			CommandJson:     dbmodel.NullString(c.CommandJSON),
			ArgsJson:        dbmodel.NullString(c.ArgsJSON),
			EnvJson:         dbmodel.NullString(c.EnvJSON),
			PortsJson:       dbmodel.NullString(c.PortsJSON),
			MountsJson:      dbmodel.NullString(c.MountsJSON),
			NetworksJson:    dbmodel.NullString(c.NetworksJSON),
			DependsOnJson:   dbmodel.NullString(c.DependsOnJSON),
			HealthcheckJson: dbmodel.NullString(c.HealthcheckJSON),
			ResourcesJson:   dbmodel.NullString(c.ResourcesJSON),
			PullPolicy:      dbmodel.NullString(c.PullPolicy),
			RestartPolicy:   dbmodel.NullString(c.RestartPolicy),
			TmpfsJson:       dbmodel.NullString(c.TmpfsJSON),
			UlimitsJson:     dbmodel.NullString(c.UlimitsJSON),
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
		}); err != nil {
			return fmt.Errorf("insert version component %s: %w", c.Name, err)
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

func (r Repository) ReplaceVersionExposes(ctx context.Context, versionId string, exposes []model.VersionExpose) error {
	q := r.q(ctx)
	if err := q.DeleteVersionExposes(ctx, versionId); err != nil {
		return fmt.Errorf("delete version exposes %s: %w", versionId, err)
	}
	now := time.Now().UTC()
	for _, e := range exposes {
		createdAt, updatedAt := e.CreatedAt, e.UpdatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		if updatedAt.IsZero() {
			updatedAt = now
		}
		if err := q.InsertVersionExpose(ctx, applicationsqlc.InsertVersionExposeParams{
			ID:            e.Id,
			VersionID:     versionId,
			ComponentName: e.ComponentName,
			Protocol:      e.Protocol,
			ContainerPort: int64(e.ContainerPort),
			PathPrefix:    dbmodel.NullString(e.PathPrefix),
			Access:        e.Access,
			ListenPort:    dbmodel.NullInt64FromIntPtr(e.ListenPort),
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		}); err != nil {
			return fmt.Errorf("insert version expose %s: %w", e.Id, err)
		}
	}
	return nil
}

func (r Repository) CreateVersionWithVersionComponentsAndExposes(ctx context.Context, version model.Version, components []model.VersionComponent, exposes []model.VersionExpose) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.CreateVersion(txCtx, version); err != nil {
			return err
		}
		if err := r.replaceVersionComponents(txCtx, version.Id, components); err != nil {
			return err
		}
		return r.ReplaceVersionExposes(txCtx, version.Id, exposes)
	})
}

func appFrom(id string, projectID sql.NullString, name, code, kind, imagePullPolicy string, createdAt, updatedAt time.Time) model.Application {
	return model.Application{
		Id:              id,
		ProjectId:       dbmodel.StringPtr(projectID),
		Name:            name,
		Code:            code,
		Kind:            kind,
		ImagePullPolicy: imagePullPolicy,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

func versionFrom(row applicationsqlc.Version) model.Version {
	return model.Version{
		Id:                   row.ID,
		ApplicationId:        row.ApplicationID,
		Label:                row.Label,
		Status:               row.Status,
		EnvJSON:              dbmodel.StringPtr(row.EnvJson),
		CreatedFromVersionId: dbmodel.StringPtr(row.CreatedFromVersionID),
		Note:                 dbmodel.StringPtr(row.Note),
		ComponentSummary:     row.ComponentSummary,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func componentFrom(row applicationsqlc.VersionComponent) model.VersionComponent {
	return model.VersionComponent{
		Id:              row.ID,
		VersionId:       row.VersionID,
		Name:            row.Name,
		Image:           row.Image,
		CommandJSON:     dbmodel.StringPtr(row.CommandJson),
		ArgsJSON:        dbmodel.StringPtr(row.ArgsJson),
		EnvJSON:         dbmodel.StringPtr(row.EnvJson),
		PortsJSON:       dbmodel.StringPtr(row.PortsJson),
		MountsJSON:      dbmodel.StringPtr(row.MountsJson),
		NetworksJSON:    dbmodel.StringPtr(row.NetworksJson),
		DependsOnJSON:   dbmodel.StringPtr(row.DependsOnJson),
		HealthcheckJSON: dbmodel.StringPtr(row.HealthcheckJson),
		ResourcesJSON:   dbmodel.StringPtr(row.ResourcesJson),
		PullPolicy:      dbmodel.StringPtr(row.PullPolicy),
		RestartPolicy:   dbmodel.StringPtr(row.RestartPolicy),
		TmpfsJSON:       dbmodel.StringPtr(row.TmpfsJson),
		UlimitsJSON:     dbmodel.StringPtr(row.UlimitsJson),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func exposeFrom(row applicationsqlc.VersionExpose) model.VersionExpose {
	return model.VersionExpose{
		Id:            row.ID,
		VersionId:     row.VersionID,
		ComponentName: row.ComponentName,
		Protocol:      row.Protocol,
		ContainerPort: int(row.ContainerPort),
		PathPrefix:    dbmodel.StringPtr(row.PathPrefix),
		Access:        row.Access,
		ListenPort:    dbmodel.IntPtrFromNullInt64(row.ListenPort),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
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
