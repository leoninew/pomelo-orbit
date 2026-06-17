package cdsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
	taskrepo "backend/internal/repository/task"
	"backend/internal/status"
)

var applicationCreateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var applicationUpdateCodePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type Store interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string) (repository.Page[repository.Application], error)
	Application(ctx context.Context, id string) (repository.Application, error)
	ApplicationByName(ctx context.Context, name string) (repository.Application, error)
	ConfigFiles(ctx context.Context, applicationId string) ([]repository.ApplicationConfigFile, error)
	CreateApplication(ctx context.Context, app repository.Application) error
	UpdateApplication(ctx context.Context, app repository.Application) error
	DeleteApplication(ctx context.Context, id string) error
	CreateDeployment(ctx context.Context, deployment repository.Deployment) error
	ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[repository.Deployment], error)
	Deployment(ctx context.Context, id string) (repository.Deployment, error)
	CancelDeployment(ctx context.Context, id string) error
}

type TaskService interface {
	EnqueueTyped(ctx context.Context, taskType string, payload any) (*taskrepo.Task, error)
}

type Service struct {
	store    Store
	tasks    TaskService
	dataRoot string
}

type ApplicationCreateInput struct {
	ProjectId       string
	Name            string
	Code            string
	ImagePullPolicy string
}

type ApplicationUpdateInput struct {
	Name            *string
	Code            *string
	ImagePullPolicy *string
	RouteManaged    *bool
}

type ApplicationDeleteInput struct {
	RemoveDir bool
}

type DeploymentListInput struct {
	ProjectId     string
	ApplicationId string
	Status        string
	Search        string
	DateFrom      string
	DateTo        string
	Page          int
	PerPage       int
}

type DeploymentLog struct {
	Logs       string
	Offset     int
	IsComplete bool
	Status     string
}

func New(store Store, tasks TaskService, dataRoot string) Service {
	return Service{store: store, tasks: tasks, dataRoot: dataRoot}
}

func (s Service) ListApplications(ctx context.Context, userId string, projectId *string, page int, perPage int, search string) (repository.Page[repository.Application], error) {
	if projectId != nil {
		if err := s.ensureProjectMembership(ctx, *projectId, userId); err != nil {
			return repository.Page[repository.Application]{}, err
		}
	}
	items, err := s.store.ListApplications(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[repository.Application]{}, apperror.Wrap(apperror.KindInternal, "Failed to list applications", err)
	}
	return items, nil
}

func (s Service) CreateApplication(ctx context.Context, userId string, input ApplicationCreateInput) (repository.Application, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Application{}, err
	}
	name, code, imagePullPolicy, err := normalizeApplicationCreateInput(input)
	if err != nil {
		return repository.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return repository.Application{}, err
	}
	app := repository.Application{Id: repository.NewId(), ProjectId: &projectId, Name: name, Code: code, ImagePullPolicy: imagePullPolicy, Status: repository.ApplicationStatusUndeployed}
	if err := s.store.CreateApplication(ctx, app); err != nil {
		return repository.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to create application", err)
	}
	created, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return repository.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return created, nil
}

func (s Service) ApplicationForUser(ctx context.Context, userId string, applicationId string) (repository.Application, error) {
	return s.loadApplicationForUser(ctx, userId, applicationId)
}

func (s Service) UpdateApplication(ctx context.Context, userId string, applicationId string, input ApplicationUpdateInput) (repository.Application, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.Application{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return repository.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.Name = name
	}
	if input.Code != nil {
		code := strings.TrimSpace(*input.Code)
		if code == "" || len(code) > 100 || !applicationUpdateCodePattern.MatchString(code) {
			return repository.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.Code = code
	}
	if input.ImagePullPolicy != nil {
		policy := strings.TrimSpace(*input.ImagePullPolicy)
		if !validImagePullPolicy(policy) {
			return repository.Application{}, apperror.New(apperror.KindValidation, "Invalid application fields")
		}
		app.ImagePullPolicy = policy
	}
	if input.RouteManaged != nil {
		app.RouteManaged = *input.RouteManaged
	}
	if err := s.store.UpdateApplication(ctx, app); err != nil {
		return repository.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to update application", err)
	}
	updated, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return repository.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return updated, nil
}

func (s Service) DeleteApplication(ctx context.Context, userId string, applicationId string, input ApplicationDeleteInput) error {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return err
	}
	if app.Status == repository.ApplicationStatusDeploying {
		return apperror.New(apperror.KindValidation, "应用正在部署中, 请稍后再试")
	}
	if app.Status == repository.ApplicationStatusDeployed {
		return apperror.New(apperror.KindValidation, "应用正在运行中, 请先停止后再删除")
	}
	if input.RemoveDir {
		if err := os.RemoveAll(filepath.Join(s.dataRoot, "cd", app.Code)); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to remove application directory", err)
		}
	}
	if err := s.store.DeleteApplication(ctx, app.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete application", err)
	}
	return nil
}

func (s Service) DeployApplication(ctx context.Context, userId string, applicationId string) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if err := s.ensureApplicationComposeFile(ctx, app.Id); err != nil {
		return "", err
	}
	deployment := newApplicationDeployment(app, "deploy")
	if err := s.store.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if _, err := s.tasks.EnqueueTyped(ctx, status.TaskTypeCDApplicationDeploy, map[string]string{"application_id": app.Id, "deployment_id": deployment.Id}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) ListDeployments(ctx context.Context, userId string, input DeploymentListInput) (repository.Page[repository.Deployment], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[repository.Deployment]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[repository.Deployment]{}, err
	}
	dateFrom, err := parseOptionalRunTime(input.DateFrom, "date_from")
	if err != nil {
		return repository.Page[repository.Deployment]{}, err
	}
	dateTo, err := parseOptionalRunTime(input.DateTo, "date_to")
	if err != nil {
		return repository.Page[repository.Deployment]{}, err
	}
	items, err := s.store.ListDeployments(ctx, projectId, input.ApplicationId, input.Status, input.Search, dateFrom, dateTo, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[repository.Deployment]{}, apperror.Wrap(apperror.KindInternal, "Failed to list deployments", err)
	}
	return items, nil
}

func (s Service) DeploymentForUser(ctx context.Context, userId string, deploymentId string) (repository.Deployment, error) {
	return s.loadDeploymentForUser(ctx, userId, deploymentId)
}

func (s Service) DeploymentLog(ctx context.Context, userId string, deploymentId string, offset int) (DeploymentLog, error) {
	if offset < 0 {
		return DeploymentLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return DeploymentLog{}, err
	}
	logs, newOffset, err := s.readDeploymentLog(ctx, deployment, offset)
	if err != nil {
		return DeploymentLog{}, err
	}
	return DeploymentLog{Logs: logs, Offset: newOffset, IsComplete: deploymentStatusComplete(deployment.Status), Status: deployment.Status}, nil
}

func (s Service) CancelDeployment(ctx context.Context, userId string, deploymentId string) (repository.Deployment, error) {
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return repository.Deployment{}, err
	}
	if deployment.Status != repository.WorkStatusWaitingToRun && deployment.Status != repository.WorkStatusRunning {
		return repository.Deployment{}, apperror.New(apperror.KindValidation, "Cannot cancel deployment with status "+deployment.Status)
	}
	if err := s.store.CancelDeployment(ctx, deployment.Id); err != nil {
		return repository.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to cancel deployment", err)
	}
	updated, err := s.store.Deployment(ctx, deployment.Id)
	if err != nil {
		return repository.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	return updated, nil
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, applicationId string) (repository.Application, error) {
	applicationId = strings.TrimSpace(applicationId)
	app, err := s.store.Application(ctx, applicationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
		}
		return repository.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
			return repository.Application{}, err
		}
	}
	return app, nil
}

func (s Service) loadDeploymentForUser(ctx context.Context, userId string, deploymentId string) (repository.Deployment, error) {
	deploymentId = strings.TrimSpace(deploymentId)
	deployment, err := s.store.Deployment(ctx, deploymentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.Deployment{}, apperror.New(apperror.KindNotFound, "Deployment "+deploymentId+" not found")
		}
		return repository.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	if deployment.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *deployment.ProjectId, userId); err != nil {
			return repository.Deployment{}, err
		}
		return deployment, nil
	}
	if deployment.ApplicationId != nil {
		app, err := s.store.Application(ctx, *deployment.ApplicationId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return repository.Deployment{}, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
			}
			return repository.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
		}
		if app.ProjectId != nil {
			if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
				return repository.Deployment{}, err
			}
		}
	}
	return deployment, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.store.Project(ctx, projectId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	if !errors.Is(err, sql.ErrNoRows) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application name", err)
	}
	return nil
}

func (s Service) ensureApplicationComposeFile(ctx context.Context, applicationId string) error {
	files, err := s.store.ConfigFiles(ctx, applicationId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load application config files", err)
	}
	for _, file := range files {
		if file.Path == "docker-compose.yml" || file.Path == "docker-compose.yml.jinja" {
			return nil
		}
	}
	return apperror.New(apperror.KindValidation, "No docker-compose file found for application "+applicationId)
}

func (s Service) readDeploymentLog(ctx context.Context, deployment repository.Deployment, offset int) (string, int, error) {
	if deployment.ApplicationId == nil || strings.TrimSpace(*deployment.ApplicationId) == "" {
		return "", offset, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated application")
	}
	app, err := s.store.Application(ctx, *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", offset, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
		}
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	logPath := filepath.Join(s.dataRoot, "cd", app.Code, "deployments", deployment.Id+".log")
	file, err := os.Open(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", offset, nil
		}
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to read deployment log", err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to read deployment log", err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to read deployment log", err)
	}
	return string(content), offset + len(content), nil
}

func normalizeApplicationCreateInput(input ApplicationCreateInput) (string, string, string, error) {
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
	return value == repository.WorkStatusRanToCompletion || value == repository.WorkStatusFaulted || value == repository.WorkStatusCanceled
}

func newApplicationDeployment(app repository.Application, operationType string) repository.Deployment {
	return repository.Deployment{Id: repository.NewId(), ProjectId: app.ProjectId, ApplicationId: &app.Id, ApplicationName: app.Name, OperationType: operationType, TriggerType: "manual", Status: repository.WorkStatusWaitingToRun, IsRollback: false}
}

func outputOrError(output string, err error) string {
	if strings.TrimSpace(output) != "" {
		return output
	}
	return fmt.Sprint(err)
}
