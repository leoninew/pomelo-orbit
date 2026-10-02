package routesvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

const (
	routeSyncPreviewExpiredCode   = "route_sync_preview_expired"
	routeSyncPublishFailedCode    = "route_sync_publish_failed"
	routeSyncPermissionDeniedCode = "route_sync_publish_permission_denied"
)

// PreviewRouteSync freezes a publication list without comparing runtime resources.
func (s Service) PreviewRouteSync(ctx context.Context, userId string, projectId string, input routedto.RouteSyncPreviewInput) (routedto.RouteSyncPreview, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return routedto.RouteSyncPreview{}, err
	}
	unlock, err := s.LockGateway(ctx, projectId)
	if err != nil {
		return routedto.RouteSyncPreview{}, err
	}
	defer unlock()
	plan, err := s.loadSyncState(ctx, userId, projectId, input)
	if err != nil {
		return routedto.RouteSyncPreview{}, err
	}
	return previewSyncPlan(plan), nil
}

// ConfirmRouteSync processes the frozen scope in order and preserves successful items.
func (s Service) ConfirmRouteSync(ctx context.Context, userId string, projectId string, input routedto.RouteSyncConfirmInput) (routedto.RouteSyncConfirmResult, error) {
	result := routedto.RouteSyncConfirmResult{Code: "route_sync_completed", Results: []routedto.RouteSyncResult{}}
	if strings.TrimSpace(input.BusinessHash) == "" || input.PublicationHash == "" {
		return result, apperror.New(apperror.KindValidation, "business_hash and publication_hash are required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(ctx, 105*time.Second)
	defer cancel()
	unlock, err := s.LockGateway(ctx, projectId)
	if err != nil {
		return result, err
	}
	defer unlock()
	plan, err := s.loadSyncState(ctx, userId, projectId, routedto.RouteSyncPreviewInput{Scope: "selected", RouteIds: input.RouteIds, Changes: input.Changes})
	if err != nil {
		return result, err
	}
	preview := previewSyncPlan(plan)
	if preview.BusinessHash != input.BusinessHash || preview.PublicationHash != input.PublicationHash {
		return result, apperror.NewWithCode(
			apperror.KindConflict,
			routeSyncPreviewExpiredCode,
			"Route sync preview is out of date. Preview the current state again.",
		)
	}
	if s.transactionRunner == nil {
		return result, apperror.New(apperror.KindInternal, "route sync transaction is not configured")
	}
	for index, route := range plan.routes {
		item := routedto.RouteSyncResult{RouteId: route.Id, RouteName: route.Name, BusinessSave: "unchanged", FileCommit: "not_attempted", ConfigurationMatch: "unverified", CertificateVerification: "not_applicable", Recovery: "not_needed", Cleanup: "not_attempted"}
		if route.Enabled && route.HTTPSEnabled {
			item.CertificateVerification = "unverified"
		}
		deadline, _ := ctx.Deadline()
		if ctx.Err() != nil || time.Until(deadline) < 2*time.Second {
			item.Code = "route_sync_skipped"
			item.Error = "The batch time budget ended before this Route was processed"
			result.Code = "route_sync_incomplete"
			result.Results = append(result.Results, item)
			continue
		}
		one := plan.routePlan(index)
		err := s.transactionRunner.RunInTransaction(ctx, func(txCtx context.Context) error { return s.verifyAndSaveSyncRoute(txCtx, projectId, one) })
		if err == nil {
			if len(one.updates) != 0 {
				item.BusinessSave = "saved"
			}
			fingerprint := ""
			if publication, found := plan.publications[route.Id]; found {
				fingerprint = hashSyncValue(publication)
			}
			if !route.Enabled && fingerprint == "" {
				item.Code = "route_sync_completed"
				item.FileCommit = "not_applicable"
				item.ConfigurationMatch = "not_applicable"
				item.Cleanup = "not_applicable"
				result.Results = append(result.Results, item)
				continue
			}
			published, publishErr := s.routePublisher.PublishRoute(ctx, projectId, plan.gateway, route, fingerprint)
			item.OperationId, item.FileCommit, item.ConfigurationMatch = published.OperationId, published.FileCommit, published.ConfigurationMatch
			item.CertificateVerification, item.Recovery, item.Cleanup = published.CertificateVerification, published.Recovery, published.Cleanup
			err = publishErr
		} else {
			item.BusinessSave = "failed"
		}
		if err != nil {
			slog.ErrorContext(ctx, "Route sync item failed", "project_id", projectId, "route_id", route.Id, "operation_id", item.OperationId, "recovery", item.Recovery, "error", err)
			item.Code, item.Error = routeSyncPublishFailedCode, "Route publication could not be completed. Preview this Route again to retry."
			if errors.Is(err, os.ErrPermission) {
				item.Code = routeSyncPermissionDeniedCode
				item.Error = "The target workspace rejected the file operation (permission denied)."
			} else if classified, ok := apperror.As(err); ok {
				if classified.Code != "" {
					item.Code = classified.Code
				}
				item.Error = classified.Message
			}
			result.Code = "route_sync_incomplete"
		} else {
			item.Code = "route_sync_completed"
		}
		result.Results = append(result.Results, item)
	}
	return result, nil
}

func (s Service) loadSyncState(ctx context.Context, userId string, projectId string, input routedto.RouteSyncPreviewInput) (syncPlan, error) {
	if projectId == "" {
		return syncPlan{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return syncPlan{}, err
	}
	gateway, err := s.resolveGatewayForRender(ctx, projectId)
	if err != nil {
		return syncPlan{}, err
	}
	publications, err := s.routePublisher.InspectPublications(ctx, projectId, *gateway)
	if err != nil {
		return syncPlan{}, err
	}
	plan, err := s.buildSyncPlan(ctx, projectId, input, *gateway, publications)
	if err != nil {
		return syncPlan{}, err
	}
	if err := s.prepareSyncRoutes(ctx, projectId, &plan); err != nil {
		return syncPlan{}, err
	}
	plan.dependencies, err = s.routePublisher.ValidateGateway(ctx, projectId, *gateway, plan.routes)
	if err != nil {
		return syncPlan{}, err
	}
	return plan, nil
}

func (s Service) prepareSyncRoutes(ctx context.Context, projectId string, plan *syncPlan) error {
	tcpPorts := make(map[int]string)
	for index := range plan.routes {
		route := &plan.routes[index]
		if !route.Enabled {
			continue
		}
		if route.Protocol != routeProtocolTCP {
			if err := s.validateRoute(ctx, projectId, route, route.Id, true); err != nil {
				return err
			}
			if err := s.validateRouteCertificateConfiguration(ctx, *route); err != nil {
				return err
			}
			continue
		}
		if route.ListenPort == nil || *route.ListenPort < 1 || *route.ListenPort > 65535 || route.ServiceId == nil || route.ComponentName == nil || route.EndpointProtocol == nil || route.EndpointContainerPort == nil {
			return apperror.New(apperror.KindValidation, "Invalid TCP route fields")
		}
		if reservedTCPRoutePort(*route.ListenPort) {
			return apperror.New(apperror.KindValidation, fmt.Sprintf("TCP listen port %d is reserved by Gateway", *route.ListenPort))
		}
		if existing, found := tcpPorts[*route.ListenPort]; found && existing != route.Name {
			return apperror.New(apperror.KindConflict, fmt.Sprintf("TCP listen port %d is already used by route %s", *route.ListenPort, existing))
		}
		if err := s.resolveManagedRouteTarget(ctx, projectId, route); err != nil {
			return err
		}
		if conflict, err := s.componentPortConflict(ctx, projectId, *route.ListenPort); err != nil {
			return err
		} else if conflict != "" {
			return apperror.New(apperror.KindConflict, fmt.Sprintf("TCP listen port %d conflicts with component endpoint %s", *route.ListenPort, conflict))
		}
		tcpPorts[*route.ListenPort] = route.Name
	}
	return nil
}
