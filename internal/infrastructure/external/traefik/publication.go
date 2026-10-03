package traefik

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/common/operation"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

// API and worker use separate managers in the same process.
var gatewayPublicationLocks sync.Map

func (m *RouteManager) LockGateway(ctx context.Context, projectId string) (func(), error) {
	value, _ := gatewayPublicationLocks.LoadOrStore(projectId, make(chan struct{}, 1))
	lock := value.(chan struct{})
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func certificateRevision(route model.Route) string {
	if !routeHasStoredCertificate(route) {
		return ""
	}
	return digest([]byte(*route.CertPEM + "\x00" + *route.CertKey))
}

func publicationFingerprint(publication routeport.Publication) string {
	data, _ := json.Marshal(publication)
	return digest(data)
}

func validPublicationId(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	for _, char := range id {
		allowed := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-'
		if !allowed {
			return false
		}
	}
	return true
}

func (m *RouteManager) publicationWorkspace(ctx context.Context, projectId string, gateway model.GatewayConfig) (environmentport.Target, deploymentport.WorkspaceFiles, string, error) {
	target, err := m.resolveTarget(ctx, projectId)
	if err != nil {
		return target, nil, "", err
	}
	files, ok := m.runtime.(deploymentport.WorkspaceFiles)
	if !ok {
		return target, nil, "", errors.New("target workspace file operations are not configured")
	}
	base, err := m.runtime.ServiceDir(target, gateway.RuntimeServiceCode)
	base = strings.ReplaceAll(base, "\\", "/")
	if err == nil {
		if session := m.session(ctx); session != nil {
			if session.base != "" && (session.base != base || session.gatewayId != gateway.ApplicationId) {
				return target, nil, "", apperror.NewWithCode(apperror.KindConflict, "route_sync_preview_expired", "Gateway publication workspace changed. Preview again.")
			}
			session.base, session.gatewayId = base, gateway.ApplicationId
		}
	}
	return target, files, base, err
}

func (m *RouteManager) ListPublicationRouteIds(ctx context.Context, projectId string, gateway model.GatewayConfig) ([]string, error) {
	ctx, closeSession, err := m.ensureSession(ctx, projectId)
	if err != nil {
		return nil, err
	}
	defer closeSession()
	target, files, base, err := m.publicationWorkspace(ctx, projectId, gateway)
	if err != nil {
		return nil, err
	}
	names, err := files.ListFiles(ctx, target, path.Join(base, ".orbit", "route-publication"))
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(names))
	for _, id := range names {
		if validPublicationId(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (m *RouteManager) InspectPublication(ctx context.Context, projectId string, gateway model.GatewayConfig, route model.Route) (item *routeport.Publication, resultErr error) {
	ctx, closeSession, err := m.ensureSession(ctx, projectId)
	if err != nil {
		return nil, err
	}
	defer closeSession()
	if route.Name != "" {
		ctx = operation.WithLogAttrs(ctx, slog.String("route_code", route.Name))
	}
	ctx, finish := operation.StartStage(ctx, "Route sync", "inspect_publication", m.timeouts.StateLoad)
	defer func() { resultErr = finish(resultErr) }()
	target, files, base, err := m.publicationWorkspace(ctx, projectId, gateway)
	if err != nil {
		return nil, err
	}
	item, err = readPublicationRecord(ctx, projectId, gateway, target, files, base, route)
	if err != nil || item == nil {
		return item, err
	}
	if err := inspectPublication(ctx, target, files, base, item); err != nil {
		return nil, err
	}
	return item, nil
}

func readPublicationRecord(ctx context.Context, projectId string, gateway model.GatewayConfig, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, route model.Route) (item *routeport.Publication, resultErr error) {
	if !validPublicationId(route.Id) {
		return nil, errors.New("invalid Route publication identity")
	}
	attrs := []any{}
	if route.Name == "" {
		attrs = append(attrs, "record_path", path.Join(base, ".orbit", "route-publication", route.Id, "state.json"))
	} else {
		ctx = operation.WithLogAttrs(ctx, slog.String("route_code", route.Name))
	}
	ctx, finish := operation.StartStage(ctx, "Route sync", "read_publication_record", 0, attrs...)
	defer func() {
		if route.Name == "" && item != nil {
			resultErr = finish(resultErr, "route_code", item.Route.Name)
		} else {
			resultErr = finish(resultErr)
		}
	}()
	data, err := files.ReadFile(ctx, target, path.Join(base, ".orbit", "route-publication", route.Id, "state.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Route %s publication record: %w", route.Name, err)
	}
	var record routeport.Publication
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("decode Route %s publication record: %w", route.Name, err)
	}
	if record.Route.Id != route.Id || record.Route.ProjectId == nil || *record.Route.ProjectId != projectId || record.GatewayApplicationId != gateway.ApplicationId || !validPublicationId(record.Route.Name) {
		return nil, errors.New("route publication ownership does not match the Gateway")
	}
	if record.Previous != nil && record.Previous.Route.Id != route.Id {
		return nil, errors.New("previous Route publication ownership does not match")
	}
	return &record, nil
}

func inspectPublication(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item *routeport.Publication) error {
	if err := inspectPublicationFile(ctx, target, files, base, item); err != nil {
		return err
	}
	if item.Phase == "pending" && item.Previous != nil {
		previous := *item.Previous
		item.Previous = &previous
		return inspectPublicationFile(ctx, target, files, base, item.Previous)
	}
	return nil
}

func (m *RouteManager) ValidateGateway(ctx context.Context, projectId string, gateway model.GatewayConfig, routes []model.Route) (fingerprint string, resultErr error) {
	ctx, finish := operation.StartStage(ctx, "Route sync", "validate_gateway", m.timeouts.StateLoad)
	defer func() { resultErr = finish(resultErr) }()
	target, err := m.resolveTarget(ctx, projectId)
	if err != nil {
		return "", err
	}
	container, err := m.gatewayContainer(ctx, target, gateway)
	if err != nil {
		return "", err
	}
	content, err := m.runtime.QueryAtEnvironmentRoot(ctx, target, "docker", "exec", container, "cat", "/etc/traefik/traefik.yml")
	if err != nil {
		return "", fmt.Errorf("inspect Gateway static configuration: %w", err)
	}
	var config struct {
		Providers struct {
			File struct {
				Directory string `yaml:"directory"`
				Watch     bool   `yaml:"watch"`
			} `yaml:"file"`
			Rest any `yaml:"rest"`
		} `yaml:"providers"`
		EntryPoints map[string]any `yaml:"entryPoints"`
		Resolvers   map[string]any `yaml:"certificatesResolvers"`
	}
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return "", err
	}
	if config.Providers.File.Directory != model.GatewayRouteConfigTarget || !config.Providers.File.Watch || config.Providers.Rest != nil {
		return "", apperror.New(apperror.KindValidation, "Gateway is not running the managed File provider configuration")
	}
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		entrypoint := "web"
		if route.Protocol == "tcp" && route.ListenPort != nil {
			entrypoint = "tcp" + strconv.Itoa(*route.ListenPort)
		} else if route.HTTPSEnabled {
			entrypoint = "websecure"
		}
		if _, found := config.EntryPoints[entrypoint]; !found {
			return "", apperror.New(apperror.KindValidation, "Gateway is missing Route entrypoint "+entrypoint)
		}
		if route.HTTPSEnabled && route.CertType == "letsencrypt" {
			resolver := "letsencrypt"
			if route.AcmeChallenge == "dns" {
				resolver = "letsencrypt-dns"
			}
			if _, found := config.Resolvers[resolver]; !found {
				return "", apperror.New(apperror.KindValidation, "Gateway is missing Route certificate resolver "+resolver)
			}
		}
	}
	output, err := m.runtime.QueryAtEnvironmentRoot(ctx, target, "docker", "inspect", container, "--format", "{{json .Mounts}}")
	if err != nil {
		return "", err
	}
	var mounts []struct {
		Source      string
		Destination string
		RW          bool
	}
	if err := json.Unmarshal([]byte(output), &mounts); err != nil {
		return "", err
	}
	physical, err := m.runtime.ComposeMountSourceDir(ctx, target, gateway.RuntimeServiceCode)
	if err != nil {
		return "", err
	}
	for _, spec := range []struct {
		source, target string
		writable       bool
	}{
		{"gateway/dynamic", model.GatewayRouteConfigTarget, false}, {"gateway/certs", "/etc/traefik/certs", false}, {"gateway/acme", "/letsencrypt", true},
	} {
		found := false
		for _, mount := range mounts {
			if mount.Destination == spec.target && strings.EqualFold(strings.ReplaceAll(mount.Source, "\\", "/"), path.Join(strings.ReplaceAll(physical, "\\", "/"), spec.source)) && mount.RW == spec.writable {
				found = true
			}
		}
		if !found {
			return "", apperror.New(apperror.KindValidation, "Gateway is missing the declared managed directory mount: "+spec.target)
		}
	}
	return digest([]byte(target.Environment.Id + ":" + strconv.FormatInt(target.Environment.TargetRevision, 10) + "\x00" + content + "\x00" + physical)), nil
}

func (m *RouteManager) gatewayContainer(ctx context.Context, target environmentport.Target, gateway model.GatewayConfig) (string, error) {
	container, err := m.runtime.Query(ctx, target, gateway.RuntimeServiceCode, "docker", "compose", "ps", "-q", model.GatewayComponentName())
	if err != nil {
		cause := fmt.Errorf("resolve running Gateway container: %w", err)
		if diagnostic := strings.TrimSpace(container); diagnostic != "" {
			const maxDiagnosticBytes = 4 * 1024
			if len(diagnostic) > maxDiagnosticBytes {
				diagnostic = "...\n" + diagnostic[len(diagnostic)-maxDiagnosticBytes:]
			}
			cause = fmt.Errorf("%w\n%s", cause, diagnostic)
		}
		return "", apperror.WrapWithCode(apperror.KindUnavailable, "route_sync_gateway_unavailable", "Failed to query the Gateway container. Check Docker, Docker Compose, and the Gateway deployment on the target host.", cause)
	}
	container = strings.TrimSpace(container)
	if container == "" || strings.ContainsAny(container, "\r\n ") {
		return "", apperror.NewWithCode(apperror.KindUnavailable, "route_sync_gateway_not_running", "The Gateway container is not running. Start or deploy the Gateway before syncing Routes.")
	}
	return container, nil
}

func (m *RouteManager) reloadFileProvider(ctx context.Context, target environmentport.Target, gateway model.GatewayConfig) (resultErr error) {
	ctx, finish := operation.StartStage(ctx, "Route sync", "reload", m.timeouts.Reload)
	defer func() { resultErr = finish(resultErr) }()
	required := target.Environment.IsSSH() && target.Environment.SSH.Platform == model.EnvironmentPlatformWindows
	if target.Environment.IsLocal() {
		physical, err := m.runtime.ComposeMountSourceDir(ctx, target, gateway.RuntimeServiceCode)
		if err != nil {
			return apperror.WrapWithCode(apperror.KindInternal, "route_sync_reload_failed", "Failed to resolve the Gateway mount source for configuration reload", err)
		}
		// DooD can expose a Windows bind mount to an Orbit process running on Linux.
		required = isWindowsMountSource(physical)
	}
	if !required {
		return nil
	}
	container, err := m.gatewayContainer(ctx, target, gateway)
	if err == nil {
		_, err = m.runtime.QueryAtEnvironmentRoot(ctx, target, "docker", "kill", "--signal=HUP", container)
	}
	if err != nil {
		return apperror.WrapWithCode(apperror.KindInternal, "route_sync_reload_failed", "Failed to reload the Traefik File provider", err)
	}
	return nil
}

func isWindowsMountSource(source string) bool {
	source = strings.ReplaceAll(source, "\\", "/")
	if len(source) >= 3 && source[1] == ':' && source[2] == '/' || strings.HasPrefix(source, "//") {
		return true
	}
	for _, prefix := range []string{"/run/desktop/mnt/host/", "/host_mnt/"} {
		if rest, ok := strings.CutPrefix(source, prefix); ok && len(rest) >= 2 && rest[1] == '/' && (rest[0] >= 'a' && rest[0] <= 'z' || rest[0] >= 'A' && rest[0] <= 'Z') {
			return true
		}
	}
	return false
}

func (m *RouteManager) loadFileConfiguration(ctx context.Context, projectId string, gateway model.GatewayConfig, target environmentport.Target, body []byte, absent ...routeResource) error {
	if err := m.reloadFileProvider(ctx, target, gateway); err != nil {
		return err
	}
	return m.waitConfiguration(ctx, projectId, gateway, body, absent...)
}

func (m *RouteManager) savePublication(ctx context.Context, target environmentport.Target, base string, item routeport.Publication) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	name := path.Join(base, ".orbit", "route-publication", item.Route.Id, "state.json")
	return m.runtime.SyncFiles(ctx, target, path.Dir(name), []deploymentport.WorkspaceFile{{Path: name, Content: data, Mode: 0o600}}, "")
}

func (m *RouteManager) PublishRoute(ctx context.Context, projectId string, gateway model.GatewayConfig, route model.Route, expectedFingerprint string) (result routeport.PublicationResult, resultErr error) {
	result = routeport.PublicationResult{OperationId: idutil.NewId(), FileCommit: "not_attempted", ConfigurationMatch: "unverified", CertificateVerification: "not_applicable", Recovery: "not_needed", Cleanup: "not_attempted"}
	if route.HTTPSEnabled && route.Enabled {
		result.CertificateVerification = "unverified"
	}
	if !validPublicationId(route.Id) || !validPublicationId(route.Name) {
		return result, errors.New("invalid Route publication identity")
	}
	ctx, closeSession, err := m.ensureSession(ctx, projectId)
	if err != nil {
		return result, err
	}
	defer closeSession()
	ctx = operation.WithLogAttrs(ctx, slog.String("route_code", route.Name))
	target, files, base, err := m.publicationWorkspace(ctx, projectId, gateway)
	if err != nil {
		return result, err
	}
	checkCtx, finishCheck := operation.StartStage(ctx, "Route sync", "verify_publication", m.timeouts.StateLoad, "operation_id", result.OperationId)
	previous, err := readPublicationRecord(checkCtx, projectId, gateway, target, files, base, route)
	if err == nil && previous != nil {
		err = inspectPublication(checkCtx, target, files, base, previous)
	}
	err = finishCheck(err)
	if err != nil {
		return result, err
	}
	actual := ""
	if previous != nil {
		actual = publicationFingerprint(*previous)
	}
	if actual != expectedFingerprint {
		return result, apperror.NewWithCode(apperror.KindConflict, "route_sync_preview_expired", "Route publication changed. Preview again.")
	}
	if previous != nil && previous.ActualFingerprint != "" && previous.ActualFingerprint != previous.Fingerprint && previous.Phase != "pending" {
		return result, apperror.NewWithCode(apperror.KindConflict, "route_sync_file_changed", "The managed Route file was modified.")
	}
	pendingVerified := false
	if previous != nil && previous.Phase == "pending" {
		recoveryCtx, stopRecovery := m.recoveryContext(ctx)
		recoveryCtx, finishRecovery := operation.StartStage(recoveryCtx, "Route sync", "pending_recovery", 0, "operation_id", result.OperationId, "timeout_ms", m.timeouts.Recovery.Milliseconds())
		previous, err = m.reconcilePending(recoveryCtx, projectId, gateway, target, files, base, *previous)
		err = finishRecovery(err)
		stopRecovery()
		if err != nil {
			result.Recovery = "failed"
			if _, classified := apperror.As(err); !classified {
				err = apperror.WrapWithCode(apperror.KindInternal, "route_sync_pending_recovery_failed", "The previous interrupted Route publication could not be confirmed or recovered", err)
			}
			return result, err
		}
		result.Recovery = "restored"
		pendingVerified = true
	}
	requestCtx := ctx
	ctx, finishFile := operation.StartStage(ctx, "Route sync", "file_publication", m.timeouts.FilePublication, "operation_id", result.OperationId)
	fileFinished := false
	completeFile := func(err error) error {
		fileFinished = true
		return finishFile(err)
	}
	defer func() {
		if !fileFinished {
			resultErr = completeFile(resultErr)
		}
	}()
	active := routeConfigurationPath(base, route)
	if route.Enabled && (previous == nil || !previous.Route.Enabled || previous.Route.Name != route.Name) {
		_, readErr := files.ReadFile(ctx, target, active)
		if readErr == nil {
			return result, apperror.New(apperror.KindConflict, "The Route file is not registered as managed")
		}
		if !os.IsNotExist(readErr) {
			return result, readErr
		}
	}
	var oldBody []byte
	if previous != nil && previous.Route.Enabled {
		oldBody, err = files.ReadFile(ctx, target, routeConfigurationPath(base, previous.Route))
		if err != nil && !os.IsNotExist(err) {
			return result, err
		}
		if err == nil && digest(oldBody) != previous.Fingerprint {
			return result, apperror.NewWithCode(apperror.KindConflict, "route_sync_file_changed", "The managed Route file was modified.")
		}
	}
	certificateFingerprint := ""
	if route.Enabled && routeHasStoredCertificate(route) {
		pair, err := tls.X509KeyPair([]byte(*route.CertPEM), []byte(*route.CertKey))
		if err != nil {
			return result, apperror.New(apperror.KindValidation, "Route certificate and private key do not form a valid pair")
		}
		certificateFingerprint = digest(pair.Certificate[0])
	}
	var body []byte
	if route.Enabled {
		body, err = yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
		if err != nil {
			return result, err
		}
	}
	cleanRoute := route
	cleanRoute.CertPEM, cleanRoute.CertKey = nil, nil
	if !route.Enabled && previous != nil {
		cleanRoute.Name = previous.Route.Name
	}
	item := routeport.Publication{Route: cleanRoute, CertificateRevision: certificateRevision(route), OperationId: result.OperationId, Phase: "pending", Previous: previous, TargetRevision: target.Environment.Id + ":" + strconv.FormatInt(target.Environment.TargetRevision, 10), GatewayApplicationId: gateway.ApplicationId}
	if previous != nil {
		copy := *previous
		copy.Previous = nil
		item.Previous = &copy
		item.PreviousFingerprint = previous.ActualFingerprint
	}
	if route.Enabled {
		item.Fingerprint = digest(body)
	}
	item.CertificateFingerprint = certificateFingerprint
	if previous != nil && publicationUnchanged(*previous, item) {
		if err := completeFile(nil); err != nil {
			return result, err
		}
		result.FileCommit = "unchanged"
		if !pendingVerified {
			matched, err := m.checkConfiguration(requestCtx, projectId, gateway, body, previous.Route.Protocol)
			if err == nil && !matched {
				err = m.loadFileConfiguration(requestCtx, projectId, gateway, target, body)
			}
			if err != nil {
				return result, err
			}
		}
		result.ConfigurationMatch = "matched"
		result.Cleanup = "not_applicable"
		return result, nil
	}
	backup := path.Join(base, ".orbit", "route-publication", route.Id, "previous.yaml")
	if previousFileAvailable(item) {
		if err := m.runtime.SyncFiles(ctx, target, path.Dir(backup), []deploymentport.WorkspaceFile{{Path: backup, Content: oldBody, Mode: 0o600}}, ""); err != nil {
			return result, err
		}
	}
	if err := m.savePublication(ctx, target, base, item); err != nil {
		return result, err
	}
	if route.Enabled && routeHasStoredCertificate(route) {
		if err := m.ensureCertificateVersion(ctx, target, files, base, route); err != nil {
			err = completeFile(err)
			result.Recovery = "restored"
			recoveryCtx, stopRecovery := m.recoveryContext(requestCtx)
			defer stopRecovery()
			recoveryCtx, finishRecovery := operation.StartStage(recoveryCtx, "Route sync", "recovery", 0, "operation_id", result.OperationId, "timeout_ms", m.timeouts.Recovery.Milliseconds())
			restoreErr := m.restoreRecord(recoveryCtx, target, base, previous, item, nil)
			restoreErr = finishRecovery(restoreErr)
			if restoreErr != nil {
				result.Recovery = "failed"
			}
			return result, errors.Join(err, restoreErr)
		}
	}
	err = m.commitPublicationFiles(ctx, target, files, base, item, body)
	err = completeFile(err)
	ctx = requestCtx
	if err == nil {
		result.FileCommit = "committed"
		err = m.loadFileConfiguration(ctx, projectId, gateway, target, body, retiredRouteResources(item)...)
	}
	if err != nil {
		result.ConfigurationMatch = "unverified"
		if classified, ok := apperror.As(err); ok && classified.Code == "route_sync_configuration_mismatch" {
			result.ConfigurationMatch = "mismatched"
		}
		result.Recovery = "failed"
		recoveryCtx, stopRecovery := m.recoveryContext(ctx)
		defer stopRecovery()
		recoveryCtx, finishRecovery := operation.StartStage(recoveryCtx, "Route sync", "recovery", 0, "operation_id", result.OperationId, "timeout_ms", m.timeouts.Recovery.Milliseconds())
		restoreErr := m.rollbackPublication(recoveryCtx, projectId, gateway, target, files, base, item, oldBody)
		restoreErr = finishRecovery(restoreErr)
		if restoreErr == nil {
			result.Recovery = "restored"
			result.FileCommit = "restored"
		}
		return result, errors.Join(err, restoreErr)
	}
	result.ConfigurationMatch = "matched"
	item.Phase = "confirmed"
	item.ActualFingerprint = item.Fingerprint
	if route.Enabled {
		item.ActualCertificateRevision = item.CertificateRevision
	}
	ctx, finish := operation.StartStage(ctx, "Route sync", "finalize_publication", m.timeouts.FilePublication, "operation_id", result.OperationId)
	defer func() { resultErr = finish(resultErr) }()
	if err := m.savePublication(ctx, target, base, item); err != nil {
		return result, err
	}
	result.Cleanup = "completed"
	if err := m.cleanupCertificateVersions(ctx, target, files, base, item); err != nil {
		result.Cleanup = "failed"
		return result, err
	}
	return result, nil
}

func (m *RouteManager) restoreRecord(ctx context.Context, target environmentport.Target, base string, previous *routeport.Publication, candidate routeport.Publication, cause error) error {
	if previous != nil {
		return errors.Join(cause, m.savePublication(ctx, target, base, *previous))
	}
	candidate.Route.Enabled = false
	candidate.Fingerprint = ""
	candidate.ActualFingerprint = ""
	candidate.Phase = "confirmed"
	candidate.Previous = nil
	return errors.Join(cause, m.savePublication(ctx, target, base, candidate))
}

// Pending records are resolved from persisted files, never from editable Route data.
func (m *RouteManager) reconcilePending(ctx context.Context, projectId string, gateway model.GatewayConfig, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication) (*routeport.Publication, error) {
	if err := validatePublicationFiles(ctx, target, files, base, item); err != nil {
		return nil, err
	}
	matched, err := publicationFilesMatch(ctx, target, files, base, item)
	if err != nil {
		return nil, err
	}
	if matched && (!item.Route.Enabled || item.ActualCertificateRevision == item.CertificateRevision) {
		var body []byte
		var err error
		if item.Route.Enabled {
			body, err = files.ReadFile(ctx, target, routeConfigurationPath(base, item.Route))
		}
		if err == nil && m.loadFileConfiguration(ctx, projectId, gateway, target, body, retiredRouteResources(item)...) == nil {
			item.Phase = "confirmed"
			item.ActualFingerprint = item.Fingerprint
			if err := m.savePublication(ctx, target, base, item); err != nil {
				return nil, err
			}
			return &item, nil
		}
	}
	var body []byte
	if previousFileAvailable(item) {
		if item.Previous.CertificateRevision != "" {
			revision, err := readCertificateRevision(ctx, target, files, base, *item.Previous)
			if err != nil {
				return nil, err
			}
			if revision != item.Previous.CertificateRevision {
				return nil, apperror.NewWithCode(apperror.KindConflict, "route_sync_certificate_changed", "Previous Route certificate files are missing or modified. Recovery material was preserved.")
			}
		}
		var err error
		body, err = files.ReadFile(ctx, target, path.Join(base, ".orbit", "route-publication", item.Route.Id, "previous.yaml"))
		if err != nil {
			return nil, err
		}
		if digest(body) != item.PreviousFingerprint {
			return nil, errors.New("previous Route recovery file does not match its publication record")
		}
	}
	if err := m.rollbackPublication(ctx, projectId, gateway, target, files, base, item, body); err != nil {
		return nil, err
	}
	if item.Previous != nil {
		item.Previous.ActualFingerprint = item.PreviousFingerprint
		return item.Previous, nil
	}
	item.Route.Enabled = false
	item.Fingerprint = ""
	item.ActualFingerprint = ""
	item.Phase = "confirmed"
	item.Previous = nil
	return &item, nil
}

func validCertificateRevision(revision string) bool {
	if len(revision) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(revision)
	return err == nil
}

func readCertificateRevision(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication) (string, error) {
	if !validCertificateRevision(item.CertificateRevision) {
		return "", errors.New("invalid Route certificate revision")
	}
	directory := path.Join(base, "gateway", "certs", model.GatewayRouteCertificateDirectory(item.Route.Id), item.CertificateRevision)
	cert, err := files.ReadFile(ctx, target, path.Join(directory, "cert.pem"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	key, err := files.ReadFile(ctx, target, path.Join(directory, "key.pem"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return digest(append(append(cert, 0), key...)), nil
}

func (m *RouteManager) ensureCertificateVersion(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, route model.Route) error {
	directory := path.Join(base, "gateway", "certs", model.GatewayRouteCertificateDirectory(route.Id), certificateRevision(route))
	missing := []deploymentport.WorkspaceFile{}
	for name, content := range map[string]string{"cert.pem": *route.CertPEM, "key.pem": *route.CertKey} {
		filename := path.Join(directory, name)
		body, err := files.ReadFile(ctx, target, filename)
		if os.IsNotExist(err) {
			missing = append(missing, deploymentport.WorkspaceFile{Path: filename, Content: []byte(content), Mode: 0o600, IgnoreIfExists: true})
			continue
		}
		if err != nil {
			return err
		}
		if string(body) != content {
			return apperror.NewWithCode(apperror.KindConflict, "route_sync_certificate_changed", "An immutable Route certificate version was modified")
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return m.runtime.SyncFiles(ctx, target, directory, missing, "")
}

func (m *RouteManager) cleanupCertificateVersions(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication) error {
	root := path.Join(base, "gateway", "certs", model.GatewayRouteCertificateDirectory(item.Route.Id))
	versions, err := files.ListFiles(ctx, target, root)
	if err != nil {
		return err
	}
	for _, revision := range versions {
		if !validCertificateRevision(revision) || revision == item.CertificateRevision || item.Previous != nil && revision == item.Previous.CertificateRevision {
			continue
		}
		for _, name := range []string{"cert.pem", "key.pem"} {
			if err := files.RemoveFile(ctx, target, path.Join(root, revision, name)); err != nil {
				return err
			}
		}
	}
	return nil
}

type dynamicRouter struct {
	Rule        string         `yaml:"rule"`
	Service     string         `yaml:"service"`
	EntryPoints []string       `yaml:"entryPoints"`
	TLS         map[string]any `yaml:"tls"`
}
type dynamicService struct {
	LoadBalancer struct {
		Servers []struct {
			URL     string `yaml:"url"`
			Address string `yaml:"address"`
		} `yaml:"servers"`
	} `yaml:"loadBalancer"`
}
type dynamicProtocol struct {
	Routers  map[string]dynamicRouter  `yaml:"routers"`
	Services map[string]dynamicService `yaml:"services"`
}
type dynamicConfiguration struct {
	HTTP dynamicProtocol `yaml:"http"`
	TCP  dynamicProtocol `yaml:"tcp"`
}

func (m *RouteManager) recoveryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), m.timeouts.Recovery)
}

func parseConfiguration(body []byte) (dynamicConfiguration, error) {
	var expected dynamicConfiguration
	if len(body) != 0 {
		if err := yaml.Unmarshal(body, &expected); err != nil {
			return expected, err
		}
		if len(expected.HTTP.Routers)+len(expected.TCP.Routers) != 1 {
			return expected, errors.New("route configuration must contain one router")
		}
	}
	return expected, nil
}

func (m *RouteManager) queryConfiguration(ctx context.Context, projectId string, gateway model.GatewayConfig, expected dynamicConfiguration, absent []routeResource, observeProtocols ...string) (bool, error) {
	protocols := map[string]bool{
		"http": len(expected.HTTP.Routers) > 0,
		"tcp":  len(expected.TCP.Routers) > 0,
	}
	for _, resource := range absent {
		protocols[resource.protocol] = true
	}
	for _, protocol := range observeProtocols {
		protocols[protocol] = true
	}
	var routers []routeport.TraefikRouter
	var services []routeport.TraefikService
	for _, protocol := range []string{"http", "tcp"} {
		if !protocols[protocol] {
			continue
		}
		observedRouters, err := m.listRouters(ctx, projectId, gateway, protocol)
		if err != nil {
			return false, err
		}
		observedServices, err := m.listServices(ctx, projectId, gateway, protocol)
		if err != nil {
			return false, err
		}
		routers = append(routers, observedRouters...)
		services = append(services, observedServices...)
	}
	return matchesConfiguration(expected, absent, routers, services), nil
}

func (m *RouteManager) checkConfiguration(ctx context.Context, projectId string, gateway model.GatewayConfig, body []byte, protocol string, absent ...routeResource) (matched bool, resultErr error) {
	ctx, finish := operation.StartStage(ctx, "Route sync", "configuration_match", m.timeouts.ConfigurationMatch)
	defer func() { resultErr = finish(resultErr) }()
	expected, err := parseConfiguration(body)
	if err != nil {
		return false, err
	}
	matched, err = m.queryConfiguration(ctx, projectId, gateway, expected, absent, protocol)
	if err != nil {
		return false, apperror.WrapWithCode(apperror.KindUnavailable, "route_sync_configuration_unavailable", "Traefik API queries failed before the Route configuration could be confirmed", err)
	}
	return matched, nil
}

func (m *RouteManager) waitConfiguration(ctx context.Context, projectId string, gateway model.GatewayConfig, body []byte, absent ...routeResource) (resultErr error) {
	ctx, finish := operation.StartStage(ctx, "Route sync", "configuration_match", m.timeouts.ConfigurationMatch)
	defer func() { resultErr = finish(resultErr) }()
	expected, err := parseConfiguration(body)
	if err != nil {
		return err
	}
	observed := false
	var lastErr error
	for {
		matched, err := m.queryConfiguration(ctx, projectId, gateway, expected, absent)
		if err == nil && matched {
			return nil
		}
		if err == nil {
			observed = true
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			if observed {
				return apperror.WrapWithCode(apperror.KindConflict, "route_sync_configuration_mismatch", "Traefik configuration did not match the committed Route", errors.Join(ctx.Err(), lastErr))
			}
			return apperror.WrapWithCode(apperror.KindUnavailable, "route_sync_configuration_unavailable", "Traefik API queries failed or timed out before the Route configuration could be confirmed", errors.Join(ctx.Err(), lastErr))
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func matchesConfiguration(expected dynamicConfiguration, absent []routeResource, routers []routeport.TraefikRouter, services []routeport.TraefikService) bool {
	for _, resource := range absent {
		routerName := model.GatewayRouteResourceName(resource.code) + "-route@file"
		serviceName := model.GatewayRouteResourceName(resource.code) + "-service@file"
		for _, router := range routers {
			if router.Protocol == resource.protocol && router.Provider == "file" && router.Name == routerName {
				return false
			}
		}
		for _, service := range services {
			if service.Protocol == resource.protocol && service.Provider == "file" && service.Name == serviceName {
				return false
			}
		}
	}
	for protocol, config := range map[string]dynamicProtocol{"http": expected.HTTP, "tcp": expected.TCP} {
		for name, desired := range config.Routers {
			found := false
			for _, router := range routers {
				var tlsConfig map[string]any
				if router.TLSConfig != "" {
					_ = json.Unmarshal([]byte(router.TLSConfig), &tlsConfig)
				}
				if tlsConfig["options"] == "default" && desired.TLS["options"] == nil {
					delete(tlsConfig, "options")
				}
				if router.Protocol == protocol && router.Provider == "file" && router.Name == name+"@file" && router.Status == "enabled" && router.Rule == desired.Rule && strings.TrimSuffix(router.Service, "@file") == desired.Service && reflect.DeepEqual(router.Entrypoints, desired.EntryPoints) && reflect.DeepEqual(tlsConfig, desired.TLS) {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		for name, desired := range config.Services {
			wanted := []string{}
			for _, server := range desired.LoadBalancer.Servers {
				if protocol == "tcp" {
					wanted = append(wanted, server.Address)
				} else {
					wanted = append(wanted, server.URL)
				}
			}
			found := false
			for _, service := range services {
				if service.Provider == "file" && service.Name == name+"@file" && service.Protocol == protocol && service.Status == "enabled" && reflect.DeepEqual(service.Servers, wanted) {
					found = true
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
