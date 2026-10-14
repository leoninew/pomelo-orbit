package deploymentsvc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sort"
	"sync"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type StatusCheckService struct {
	project        repository.ProjectReader
	services       deploymentport.StatusCheckStore
	targetResolver environmentport.TargetResolver
	inventory      deploymentport.ContainerInventory
	checks         *serviceStatusChecks
}

func NewStatusCheckService(project repository.ProjectReader, services deploymentport.StatusCheckStore, targetResolver environmentport.TargetResolver, inventory deploymentport.ContainerInventory) *StatusCheckService {
	return &StatusCheckService{project: project, services: services, targetResolver: targetResolver, inventory: inventory, checks: newServiceStatusChecks()}
}

const serviceStatusCheckLifetime = 15 * time.Minute

type serviceStatusCheckScope struct {
	actorId   string
	projectId string
}

type serviceStatusObservation struct {
	service model.Service
	status  string
}

type serviceStatusCheckSnapshot struct {
	token          string
	environmentId  string
	targetRevision int64
	expiresAt      time.Time
	observations   map[string]serviceStatusObservation
}

// Only the latest observation per actor and Project can be confirmed.
// Snapshots are immutable after publication and never contain SSH credentials.
type serviceStatusChecks struct {
	mu        sync.Mutex
	snapshots map[serviceStatusCheckScope]serviceStatusCheckSnapshot
}

func newServiceStatusChecks() *serviceStatusChecks {
	return &serviceStatusChecks{snapshots: make(map[serviceStatusCheckScope]serviceStatusCheckSnapshot)}
}

func (s *serviceStatusChecks) prune() {
	for scope, snapshot := range s.snapshots {
		if !time.Now().Before(snapshot.expiresAt) {
			delete(s.snapshots, scope)
		}
	}
}

func (s *StatusCheckService) PreviewServiceStatusCheck(ctx context.Context, actorId, projectId string) (deploymentdto.ServiceStatusCheckPreview, error) {
	if projectId == "" {
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := ensureProjectMembership(ctx, s.project, projectId, actorId); err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, err
	}
	services, err := s.services.DirectoryServices(ctx, projectId)
	if err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	target, err := s.targetResolver.ResolveProjectTarget(ctx, projectId)
	if err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, err
	}
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	containers, err := s.inventory.Containers(queryCtx, target)
	if err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.WrapWithCode(apperror.KindUnavailable, "service_status_check_unavailable", "Failed to query environment containers", err)
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.Wrap(apperror.KindInternal, "Failed to create status check snapshot", err)
	}
	now := time.Now().UTC()
	preview := deploymentdto.ServiceStatusCheckPreview{
		Token: base64.RawURLEncoding.EncodeToString(secret[:]), ObservedAt: now,
		Items: make([]deploymentdto.ServiceStatusCheckItem, 0, len(services)),
	}
	snapshot := serviceStatusCheckSnapshot{
		token: preview.Token, environmentId: target.Environment.Id, targetRevision: target.Environment.TargetRevision,
		expiresAt: now.Add(serviceStatusCheckLifetime), observations: make(map[string]serviceStatusObservation, len(services)),
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Code < services[j].Code })
	for _, service := range services {
		observed := inventoryServiceStatus(containers[service.Code])
		preview.Items = append(preview.Items, deploymentdto.ServiceStatusCheckItem{
			ServiceId: service.Id, ServiceCode: service.Code, RecordedStatus: service.Status, ObservedStatus: observed,
		})
		snapshot.observations[service.Id] = serviceStatusObservation{service: service, status: observed}
	}
	s.checks.mu.Lock()
	defer s.checks.mu.Unlock()
	s.checks.prune()
	s.checks.snapshots[serviceStatusCheckScope{actorId: actorId, projectId: projectId}] = snapshot
	return preview, nil
}

func (s *StatusCheckService) ConfirmServiceStatusCheck(ctx context.Context, actorId, projectId, token, serviceId string) (string, error) {
	if projectId == "" || token == "" || serviceId == "" {
		return "", apperror.New(apperror.KindValidation, "project_id, token and service_id are required")
	}
	if err := ensureProjectMembership(ctx, s.project, projectId, actorId); err != nil {
		return "", err
	}
	s.checks.mu.Lock()
	s.checks.prune()
	snapshot, found := s.checks.snapshots[serviceStatusCheckScope{actorId: actorId, projectId: projectId}]
	s.checks.mu.Unlock()
	if !found || snapshot.token != token {
		return "", apperror.NewWithCode(apperror.KindConflict, "service_status_check_expired", "Status check expired; check again")
	}
	observation, found := snapshot.observations[serviceId]
	if !found {
		return "", apperror.New(apperror.KindValidation, "Service is outside this status check")
	}
	updated, err := s.services.UpdateServiceObservedStatus(ctx, observation.service, snapshot.environmentId, snapshot.targetRevision, observation.status)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to update service status", err)
	}
	if !updated {
		return "", apperror.NewWithCode(apperror.KindConflict, "service_status_check_changed", "Service or environment changed, or a deployment is active; check again after it finishes")
	}
	return observation.status, nil
}

func inventoryServiceStatus(containers []deploymentport.ContainerState) string {
	if len(containers) == 0 {
		return status.ServiceStatusStopped
	}
	running, stopped := 0, 0
	for _, container := range containers {
		switch container.State {
		case "running":
			if container.Unhealthy {
				return status.ServiceStatusFaulted
			}
			running++
		case "exited", "created":
			stopped++
		default:
			return status.ServiceStatusFaulted
		}
	}
	if running == len(containers) {
		return status.ServiceStatusRunning
	}
	if stopped == len(containers) {
		return status.ServiceStatusStopped
	}
	return status.ServiceStatusFaulted
}
