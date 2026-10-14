package deploymentsvc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

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

func (s Service) PreviewServiceStatusCheck(ctx context.Context, actorId, projectId string) (deploymentdto.ServiceStatusCheckPreview, error) {
	if projectId == "" {
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, actorId); err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, err
	}
	services, err := s.service.DirectoryServices(ctx, projectId)
	if err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
	}
	target, err := s.targetResolver.ResolveProjectTarget(ctx, projectId)
	if err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, err
	}
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	const format = `{{json .}}`
	output, err := s.runtime.QueryAtEnvironmentRoot(queryCtx, target, "docker", "ps", "--all", "--filter", "label=com.docker.compose.project", "--format", format)
	if err != nil {
		if diagnostic := strings.TrimSpace(output); diagnostic != "" {
			const maxDiagnosticBytes = 8192
			if len(diagnostic) > maxDiagnosticBytes {
				diagnostic = "...\n" + diagnostic[len(diagnostic)-maxDiagnosticBytes:]
			}
			err = fmt.Errorf("%w\n%s", err, diagnostic)
		}
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.WrapWithCode(apperror.KindUnavailable, "service_status_check_unavailable", "Failed to query environment containers", err)
	}
	containers, err := parseServiceContainerInventory(output)
	if err != nil {
		return deploymentdto.ServiceStatusCheckPreview{}, apperror.Wrap(apperror.KindUnavailable, "Failed to read environment container states", err)
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
	s.statusChecks.mu.Lock()
	defer s.statusChecks.mu.Unlock()
	s.statusChecks.prune()
	s.statusChecks.snapshots[serviceStatusCheckScope{actorId: actorId, projectId: projectId}] = snapshot
	return preview, nil
}

func (s Service) ConfirmServiceStatusCheck(ctx context.Context, actorId, projectId, token, serviceId string) (string, error) {
	if projectId == "" || token == "" || serviceId == "" {
		return "", apperror.New(apperror.KindValidation, "project_id, token and service_id are required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, actorId); err != nil {
		return "", err
	}
	s.statusChecks.mu.Lock()
	s.statusChecks.prune()
	snapshot, found := s.statusChecks.snapshots[serviceStatusCheckScope{actorId: actorId, projectId: projectId}]
	s.statusChecks.mu.Unlock()
	if !found || snapshot.token != token {
		return "", apperror.NewWithCode(apperror.KindConflict, "service_status_check_expired", "Status check expired; check again")
	}
	observation, found := snapshot.observations[serviceId]
	if !found {
		return "", apperror.New(apperror.KindValidation, "Service is outside this status check")
	}
	updated, err := s.service.UpdateServiceObservedStatus(ctx, observation.service, snapshot.environmentId, snapshot.targetRevision, observation.status)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to update service status", err)
	}
	if !updated {
		return "", apperror.NewWithCode(apperror.KindConflict, "service_status_check_changed", "Service or environment changed, or a deployment is active; check again after it finishes")
	}
	return observation.status, nil
}

type serviceInventoryContainer struct {
	Labels string
	State  string
	Status string
}

func parseServiceContainerInventory(output string) (map[string][]serviceInventoryContainer, error) {
	result := make(map[string][]serviceInventoryContainer)
	decoder := json.NewDecoder(strings.NewReader(output))
	for {
		var container serviceInventoryContainer
		if err := decoder.Decode(&container); err != nil {
			if errors.Is(err, io.EOF) {
				return result, nil
			}
			return nil, err
		}
		project, oneoff := "", ""
		for _, label := range strings.Split(container.Labels, ",") {
			key, value, _ := strings.Cut(label, "=")
			switch key {
			case "com.docker.compose.project":
				project = value
			case "com.docker.compose.oneoff":
				oneoff = value
			}
		}
		if project == "" || strings.EqualFold(oneoff, "true") {
			continue
		}
		result[project] = append(result[project], container)
	}
}

func inventoryServiceStatus(containers []serviceInventoryContainer) string {
	if len(containers) == 0 {
		return status.ServiceStatusStopped
	}
	running, stopped := 0, 0
	for _, container := range containers {
		switch container.State {
		case "running":
			if strings.Contains(container.Status, "(unhealthy)") {
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
