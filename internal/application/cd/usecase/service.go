package cdsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"log/slog"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	cdport "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var applicationCreateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var applicationUpdateCodePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type Service struct {
	store                repository.CDStore
	executionStore       repository.DeploymentExecutionStore
	dispatcher           cdport.ApplicationDispatcher
	cfg                  config.Config
	workspace            cdport.Workspace
	logStore             cdport.LogReader
	executionLogStore    cdport.ExecutionLogStore
	logger               *slog.Logger
	runner               cdport.CommandRunner
	queryRunner          cdport.CommandQueryRunner
	routePublisher       cdport.RouteConfigPublisher
	certificateGenerator cdport.RouteCertificateGenerator
	traefikRouterClient  cdport.TraefikRouterClient
}

func New(store repository.CDStore, dispatcher cdport.ApplicationDispatcher, cfg config.Config, logger *slog.Logger, workspace cdport.Workspace, queryRunner cdport.CommandQueryRunner, logStore cdport.LogReader, routePublisher cdport.RouteConfigPublisher, certificateGenerator cdport.RouteCertificateGenerator, traefikRouterClient cdport.TraefikRouterClient) Service {
	executionStore, ok := store.(repository.DeploymentExecutionStore)
	if !ok {
		panic("cd service store must implement DeploymentExecutionStore")
	}
	return Service{store: store, executionStore: executionStore, dispatcher: dispatcher, cfg: cfg, workspace: workspace, queryRunner: queryRunner, logStore: logStore, logger: logger, routePublisher: routePublisher, certificateGenerator: certificateGenerator, traefikRouterClient: traefikRouterClient}
}

func NewExecutionService(store repository.DeploymentExecutionStore, cfg config.Config, logger *slog.Logger, workspace cdport.Workspace, runner cdport.CommandRunner, logStore cdport.ExecutionLogStore) Service {
	return Service{executionStore: store, cfg: cfg, workspace: workspace, logStore: logStore, executionLogStore: logStore, logger: logger, runner: runner}
}

func (s Service) ListApplications(ctx context.Context, userId string, projectId *string, page int, perPage int, search string) (repository.Page[model.Application], error) {
	if projectId != nil {
		if err := s.ensureProjectMembership(ctx, *projectId, userId); err != nil {
			return repository.Page[model.Application]{}, err
		}
	}
	items, err := s.store.ListApplications(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Application]{}, apperror.Wrap(apperror.KindInternal, "Failed to list applications", err)
	}
	return items, nil
}

func (s Service) CreateApplication(ctx context.Context, userId string, input cdto.ApplicationCreateInput) (model.Application, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Application{}, err
	}
	name, code, imagePullPolicy, err := normalizeApplicationCreateInput(input)
	if err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return model.Application{}, err
	}
	app := model.Application{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, ImagePullPolicy: imagePullPolicy}
	if err := s.store.CreateApplication(ctx, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to create application", err)
	}
	created, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return created, nil
}

func (s Service) ApplicationForUser(ctx context.Context, userId string, applicationId string) (model.Application, error) {
	return s.loadApplicationForUser(ctx, userId, applicationId)
}

func (s Service) UpdateApplication(ctx context.Context, userId string, applicationId string, input cdto.ApplicationUpdateInput) (model.Application, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.Application{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.Name = name
	}
	if input.Code != nil {
		code := strings.TrimSpace(*input.Code)
		if code == "" || len(code) > 100 || !applicationUpdateCodePattern.MatchString(code) {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.Code = code
	}
	if input.ImagePullPolicy != nil {
		policy := strings.TrimSpace(*input.ImagePullPolicy)
		if !validImagePullPolicy(policy) {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.ImagePullPolicy = policy
	}
	if err := s.store.UpdateApplication(ctx, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to update application", err)
	}
	updated, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return updated, nil
}

func (s Service) DeleteApplication(ctx context.Context, userId string, applicationId string, input cdto.ApplicationDeleteInput) error {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return err
	}
	services, err := s.store.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load services", err)
	}
	for _, svc := range services {
		if svc.Status == status.ServiceStatusDeploying {
			return apperror.New(apperror.KindValidation, "应用正在部署中, 请稍后再试")
		}
		if svc.Status == status.ServiceStatusRunning {
			return apperror.New(apperror.KindValidation, "应用正在运行中, 请先停止后再删除")
		}
	}
	if input.RemoveDir {
		if err := s.workspace.RemoveAppDir(app.Code); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to remove application directory", err)
		}
	}
	if err := s.store.DeleteApplication(ctx, app.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete application", err)
	}
	return nil
}

func (s Service) DeployApplication(ctx context.Context, userId string, applicationId string, input cdto.ApplicationDeployInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	versionId := strings.TrimSpace(input.VersionId)
	if versionId == "" {
		return "", apperror.New(apperror.KindValidation, "version_id is required")
	}
	environmentId := strings.TrimSpace(input.EnvironmentId)
	if environmentId == "" {
		return "", apperror.New(apperror.KindValidation, "environment_id is required")
	}
	version, err := s.store.Version(ctx, versionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Version "+versionId+" not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	if version.ApplicationId != app.Id {
		return "", apperror.New(apperror.KindValidation, "Version does not belong to this application")
	}
	env, err := s.store.Environment(ctx, environmentId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Environment not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	if app.ProjectId == nil || *app.ProjectId != env.ProjectId {
		return "", apperror.New(apperror.KindValidation, "application and environment must belong to the same project")
	}
	components, err := s.store.ComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if err := validateComponents(components); err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	instanceKey := strings.TrimSpace(input.InstanceKey)
	if instanceKey == "" {
		instanceKey = "default"
	}
	attachIngress := instanceKey == "default"
	if input.AttachIngress != nil {
		attachIngress = *input.AttachIngress
	}
	deployment := newApplicationDeployment(app, "deploy")
	deployment.VersionId = &version.Id
	deployment.EnvironmentId = &env.Id
	opts := cdto.DeployOptionsJSON{
		ForceRecreate: input.ForceRecreate,
		InstanceKey:   instanceKey,
		AttachIngress: attachIngress,
	}
	raw, _ := json.Marshal(opts)
	text := string(raw)
	deployment.OptionsJSON = &text
	if svc, err := s.store.ServiceByKey(ctx, app.Id, env.Id, instanceKey); err == nil {
		deployment.ServiceId = &svc.Id
	} else if !errors.Is(err, repository.ErrNotFound) {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	deployment.CommandText = deployComposeCommand(composeProjectName(app.Code, env.Code, instanceKey), app.ImagePullPolicy, input.ForceRecreate).String()
	if err := s.store.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchApplicationDeploy(ctx, cdto.ApplicationDeployDispatchInput{ApplicationID: app.Id, DeploymentID: deployment.Id, ForceRecreate: input.ForceRecreate}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) ListDeployments(ctx context.Context, userId string, input cdto.DeploymentListInput) (repository.Page[model.Deployment], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[model.Deployment]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Deployment]{}, err
	}
	dateFrom, err := parseOptionalRunTime(input.DateFrom, "date_from")
	if err != nil {
		return repository.Page[model.Deployment]{}, err
	}
	dateTo, err := parseOptionalRunTime(input.DateTo, "date_to")
	if err != nil {
		return repository.Page[model.Deployment]{}, err
	}
	items, err := s.store.ListDeployments(ctx, projectId, input.ApplicationId, input.Status, input.Search, dateFrom, dateTo, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[model.Deployment]{}, apperror.Wrap(apperror.KindInternal, "Failed to list deployments", err)
	}
	return items, nil
}

func (s Service) DeploymentForUser(ctx context.Context, userId string, deploymentId string) (model.Deployment, error) {
	return s.loadDeploymentForUser(ctx, userId, deploymentId)
}

func (s Service) DeploymentLog(ctx context.Context, userId string, deploymentId string, offset int) (cdto.DeploymentLog, error) {
	if offset < 0 {
		return cdto.DeploymentLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return cdto.DeploymentLog{}, err
	}
	logs, newOffset, err := s.readDeploymentLog(ctx, deployment, offset)
	if err != nil {
		return cdto.DeploymentLog{}, err
	}
	return cdto.DeploymentLog{Logs: logs, Offset: newOffset, IsComplete: deploymentStatusComplete(deployment.Status), Status: deployment.Status}, nil
}

func (s Service) CancelDeployment(ctx context.Context, userId string, deploymentId string) (model.Deployment, error) {
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return model.Deployment{}, err
	}
	if deployment.Status != status.WorkStatusWaitingToRun && deployment.Status != status.WorkStatusRunning {
		return model.Deployment{}, apperror.New(apperror.KindValidation, "Cannot cancel deployment with status "+deployment.Status)
	}
	if err := s.store.CancelDeployment(ctx, deployment.Id); err != nil {
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to cancel deployment", err)
	}
	updated, err := s.store.Deployment(ctx, deployment.Id)
	if err != nil {
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	return updated, nil
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, applicationId string) (model.Application, error) {
	applicationId = strings.TrimSpace(applicationId)
	app, err := s.store.Application(ctx, applicationId)
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

func (s Service) loadDeploymentForUser(ctx context.Context, userId string, deploymentId string) (model.Deployment, error) {
	deploymentId = strings.TrimSpace(deploymentId)
	deployment, err := s.store.Deployment(ctx, deploymentId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Deployment{}, apperror.New(apperror.KindNotFound, "Deployment "+deploymentId+" not found")
		}
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	if deployment.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *deployment.ProjectId, userId); err != nil {
			return model.Deployment{}, err
		}
		return deployment, nil
	}
	if deployment.ApplicationId != nil {
		app, err := s.store.Application(ctx, *deployment.ApplicationId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return model.Deployment{}, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
			}
			return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
		}
		if app.ProjectId != nil {
			if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
				return model.Deployment{}, err
			}
		}
	}
	return deployment, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.store.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.store.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func (s Service) ensureApplicationNameAvailable(ctx context.Context, name string) error {
	existing, err := s.store.ApplicationByName(ctx, name)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application '"+existing.Name+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application name", err)
	}
	return nil
}

func (s Service) readDeploymentLog(ctx context.Context, deployment model.Deployment, offset int) (string, int, error) {
	if deployment.ApplicationId == nil || strings.TrimSpace(*deployment.ApplicationId) == "" {
		return "", offset, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated application")
	}
	app, err := s.store.Application(ctx, *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", offset, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
		}
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	envCode := "local"
	instanceKey := "default"
	if deployment.EnvironmentId != nil && strings.TrimSpace(*deployment.EnvironmentId) != "" {
		if env, err := s.store.Environment(ctx, *deployment.EnvironmentId); err == nil {
			envCode = env.Code
		}
	}
	opts := parseDeployOptions(deployment.OptionsJSON)
	if strings.TrimSpace(opts.InstanceKey) != "" {
		instanceKey = opts.InstanceKey
	}
	logPath := s.workspace.DeploymentLogPath(app.Code, envCode, instanceKey, deployment.Id)
	content, newOffset, err := s.logStore.Read(logPath, offset)
	if err != nil {
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to read deployment log", err)
	}
	return string(content), newOffset, nil
}

func normalizeApplicationCreateInput(input cdto.ApplicationCreateInput) (string, string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	imagePullPolicy := strings.TrimSpace(input.ImagePullPolicy)
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !applicationCreateCodePattern.MatchString(code) || !validImagePullPolicy(imagePullPolicy) {
		return "", "", "", apperror.New(apperror.KindValidation, "Invalid application fields")
	}
	return name, code, imagePullPolicy, nil
}

func validImagePullPolicy(value string) bool {
	return value == "always" || value == "missing" || value == "never"
}

func parseOptionalRunTime(value string, name string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, name+" must be ISO 8601")
	}
	return &parsed, nil
}

func deploymentStatusComplete(value string) bool {
	return value == status.WorkStatusRanToCompletion || value == status.WorkStatusFaulted || value == status.WorkStatusCanceled
}

func newApplicationDeployment(app model.Application, operationType string) model.Deployment {
	return model.Deployment{Id: idutil.NewId(), ProjectId: app.ProjectId, ApplicationId: &app.Id, ApplicationName: app.Name, OperationType: operationType, TriggerType: "manual", Status: status.WorkStatusWaitingToRun, IsRollback: false}
}

func outputOrError(output string, err error) string {
	if strings.TrimSpace(output) != "" {
		return output
	}
	return fmt.Sprint(err)
}
