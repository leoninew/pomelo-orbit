package servicesvc

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type applicationReader interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Version(ctx context.Context, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
}

type credentialReader interface {
	Credential(ctx context.Context, id string) (model.Credential, error)
}

type serviceStore interface {
	repository.ServiceReader
	DeleteService(ctx context.Context, id string) error
}

type Service struct {
	project     repository.ProjectReader
	application applicationReader
	credential  credentialReader
	secretKey   string
	service     serviceStore
}

func New(project repository.ProjectReader, application applicationReader, credential credentialReader, secretKey string, service serviceStore) Service {
	return Service{
		project: project, application: application, credential: credential, secretKey: secretKey, service: service,
	}
}

// DeleteService removes a stopped runtime binding while preserving its deployment history.
func (s Service) DeleteService(ctx context.Context, userId string, serviceId string) error {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return apperror.New(apperror.KindValidation, "service_id is required")
	}
	item, err := s.service.ServiceListItem(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, item.ApplicationId); err != nil {
		return err
	}
	if item.Status != status.ServiceStatusStopped {
		return apperror.New(apperror.KindValidation, "Only stopped services can be deleted")
	}
	if err := s.service.DeleteService(ctx, item.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete service", err)
	}
	return nil
}

// ListServices returns runtime bindings for a project.
func (s Service) ListServices(ctx context.Context, userId string, input servicedto.ServiceListInput) (repository.Page[servicedto.ServiceView], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[servicedto.ServiceView]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[servicedto.ServiceView]{}, err
	}
	page, err := s.service.ListServicesByProject(ctx, projectId, input.ApplicationId, input.Status, input.Search, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[servicedto.ServiceView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	items := make([]servicedto.ServiceView, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, serviceViewFromListItem(item))
	}
	return repository.Page[servicedto.ServiceView]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}, nil
}

// GetService returns one runtime binding with display labels when the user can access its application.
func (s Service) GetService(ctx context.Context, userId string, serviceId string) (servicedto.ServiceView, error) {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return servicedto.ServiceView{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	item, err := s.service.ServiceListItem(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return servicedto.ServiceView{}, apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
		}
		return servicedto.ServiceView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if _, err := s.loadApplicationForUser(ctx, userId, item.ApplicationId); err != nil {
		return servicedto.ServiceView{}, err
	}
	return serviceViewFromListItem(item), nil
}

// RuntimeEnv returns the persisted credential values referenced by the service's current Version.
func (s Service) RuntimeEnv(ctx context.Context, userId string, serviceId string) (servicedto.RuntimeEnvView, error) {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return servicedto.RuntimeEnvView{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	item, err := s.service.ServiceListItem(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return servicedto.RuntimeEnvView{}, apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
		}
		return servicedto.RuntimeEnvView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	app, err := s.loadApplicationForUser(ctx, userId, item.ApplicationId)
	if err != nil {
		return servicedto.RuntimeEnvView{}, err
	}
	version, err := s.application.Version(ctx, item.VersionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return servicedto.RuntimeEnvView{}, apperror.New(apperror.KindNotFound, "Version "+item.VersionId+" not found")
		}
		return servicedto.RuntimeEnvView{}, apperror.Wrap(apperror.KindInternal, "Failed to load service version", err)
	}
	if version.ApplicationId != app.Id {
		return servicedto.RuntimeEnvView{}, apperror.New(apperror.KindNotFound, "Service "+serviceId+" not found")
	}
	components, err := s.application.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return servicedto.RuntimeEnvView{}, apperror.Wrap(apperror.KindInternal, "Failed to load version components", err)
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Name < components[j].Name })

	items := make([]servicedto.RuntimeEnvItem, 0)
	for _, component := range components {
		refs := append([]model.VersionComponentSecretEnvRef(nil), component.SecretEnvRefs...)
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].EnvKey != refs[j].EnvKey {
				return refs[i].EnvKey < refs[j].EnvKey
			}
			if refs[i].CredentialId != refs[j].CredentialId {
				return refs[i].CredentialId < refs[j].CredentialId
			}
			return refs[i].DataKey < refs[j].DataKey
		})
		for _, ref := range refs {
			credential, err := s.credential.Credential(ctx, ref.CredentialId)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return servicedto.RuntimeEnvView{}, apperror.New(apperror.KindNotFound, "Credential "+ref.CredentialId+" not found")
				}
				return servicedto.RuntimeEnvView{}, apperror.Wrap(apperror.KindInternal, "Failed to load runtime_env credential", err)
			}
			if credential.Type != "runtime_env" || app.ProjectId == nil || credential.ProjectId == nil || *credential.ProjectId != *app.ProjectId {
				return servicedto.RuntimeEnvView{}, apperror.New(apperror.KindInternal, "Invalid runtime_env credential reference")
			}
			value, err := s.runtimeEnvCredentialValue(credential.EncryptedData, ref.DataKey)
			if err != nil {
				return servicedto.RuntimeEnvView{}, apperror.Wrap(apperror.KindInternal, "Failed to read runtime_env credential", err)
			}
			items = append(items, servicedto.RuntimeEnvItem{
				ComponentName: component.Name, EnvKey: ref.EnvKey, CredentialId: credential.Id,
				CredentialName: credential.Name, DataKey: ref.DataKey, Value: value,
			})
		}
	}
	return servicedto.RuntimeEnvView{ServiceId: item.Id, VersionId: version.Id, Items: items}, nil
}

func (s Service) runtimeEnvCredentialValue(encryptedData string, dataKey string) (string, error) {
	if s.secretKey == "" {
		return "", errors.New("runtime_env credential decryption is not configured")
	}
	plain, err := security.DecryptString(s.secretKey, encryptedData)
	if err != nil {
		return "", err
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(plain), &values); err != nil || len(values) == 0 {
		if err != nil {
			return "", err
		}
		return "", errors.New("runtime_env credential data is empty")
	}
	value, exists := values[dataKey]
	if !exists {
		return "", errors.New("runtime_env credential data_key is missing")
	}
	return value, nil
}

// ListServicesByApplication returns runtime bindings after authorizing access to the application.
func (s Service) ListServicesByApplication(ctx context.Context, userId string, applicationId string) ([]model.Service, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	services, err := s.service.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	return services, nil
}

// PrimaryServiceByApplication returns the first runtime binding, if the application is running.
func (s Service) PrimaryServiceByApplication(ctx context.Context, userId string, applicationId string) (*model.Service, error) {
	services, err := s.ListServicesByApplication(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, nil
	}
	return &services[0], nil
}

// ResolveServiceTarget authorizes the application and resolves an unambiguous runtime binding.
func (s Service) ResolveServiceTarget(ctx context.Context, userId string, applicationId string, input servicedto.ServiceTargetInput) (model.Service, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.Service{}, err
	}
	return s.resolveServiceTarget(ctx, app.Id, input)
}

func (s Service) resolveServiceTarget(ctx context.Context, applicationId string, input servicedto.ServiceTargetInput) (model.Service, error) {
	if serviceId := strings.TrimSpace(input.ServiceId); serviceId != "" {
		svc, err := s.service.Service(ctx, serviceId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
			}
			return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
		}
		if svc.ApplicationId != applicationId {
			return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return svc, nil
	}

	if input.InstanceKey == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "instance_key is required")
	}

	svc, err := s.service.ServiceByKey(ctx, applicationId, input.InstanceKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, apperror.New(apperror.KindValidation, "应用未在运行中")
		}
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return svc, nil
}

func serviceViewFromListItem(item model.ServiceListItem) servicedto.ServiceView {
	return servicedto.ServiceView{
		Service:                    item.Service(),
		ApplicationName:            item.ApplicationName,
		ApplicationCode:            item.ApplicationCode,
		ApplicationKind:            item.ApplicationKind,
		VersionLabel:               item.VersionLabel,
		LastSuccessfulVersionLabel: item.LastSuccessfulVersionLabel,
	}
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, applicationId string) (model.Application, error) {
	applicationId = strings.TrimSpace(applicationId)
	app, err := s.application.Application(ctx, applicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
			return model.Application{}, err
		}
	}
	return app, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}
