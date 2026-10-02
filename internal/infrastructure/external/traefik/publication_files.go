package traefik

import (
	"context"
	"errors"
	"os"
	"path"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func routeConfigurationPath(base string, route model.Route) string {
	return path.Join(base, "gateway", "dynamic", model.GatewayRouteResourceName(route.Name)+".yaml")
}

func readFileFingerprint(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, name string) (string, error) {
	body, err := files.ReadFile(ctx, target, name)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return digest(body), nil
}

func inspectPublicationFile(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item *routeport.Publication) error {
	item.ActualFingerprint = ""
	item.ActualCertificateRevision = ""
	if !validPublicationId(item.Route.Name) {
		return errors.New("invalid Route publication code")
	}
	if !item.Route.Enabled {
		return nil
	}
	var err error
	item.ActualFingerprint, err = readFileFingerprint(ctx, target, files, routeConfigurationPath(base, item.Route))
	if err == nil && item.CertificateRevision != "" {
		item.ActualCertificateRevision, err = readCertificateRevision(ctx, target, files, base, *item)
	}
	return err
}

func validateRouteCodeOwnership(route model.Route, publications []routeport.Publication) error {
	if !route.Enabled {
		return nil
	}
	for _, published := range publications {
		if published.Route.Id == route.Id {
			continue
		}
		owners := []routeport.Publication{published}
		if published.Phase == "pending" && published.Previous != nil {
			owners = append(owners, *published.Previous)
		}
		for _, owner := range owners {
			if owner.Route.Enabled && owner.Route.Name == route.Name {
				return apperror.New(apperror.KindConflict, "The Route code is still owned by another publication. Withdraw or rename it first.")
			}
		}
	}
	return nil
}

func retiredRouteCodes(item routeport.Publication) []string {
	if item.Previous != nil && item.Previous.Route.Enabled && (!item.Route.Enabled || item.Route.Name != item.Previous.Route.Name) {
		return []string{item.Previous.Route.Name}
	}
	return nil
}

func previousFileAvailable(item routeport.Publication) bool {
	return item.Previous != nil && item.Previous.Route.Enabled && item.PreviousFingerprint != ""
}

// A rename tracks both paths; missing files can be an interrupted commit,
// but an unrelated body must never be overwritten during recovery.
func validatePublicationFiles(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication) error {
	allowed := make(map[string][]string)
	if item.Route.Enabled {
		allowed[routeConfigurationPath(base, item.Route)] = []string{item.Fingerprint, ""}
	}
	if item.Previous != nil && item.Previous.Route.Enabled {
		name := routeConfigurationPath(base, item.Previous.Route)
		allowed[name] = append(allowed[name], item.Previous.Fingerprint, "")
	}
	for name, fingerprints := range allowed {
		actual, err := readFileFingerprint(ctx, target, files, name)
		if err != nil {
			return err
		}
		known := false
		for _, fingerprint := range fingerprints {
			known = known || actual == fingerprint
		}
		if !known {
			return apperror.NewWithCode(apperror.KindConflict, "route_sync_file_changed", "Recovery preserved an unknown external file change")
		}
	}
	return nil
}

func publicationFilesMatch(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication) (bool, error) {
	if item.Route.Enabled {
		actual, err := readFileFingerprint(ctx, target, files, routeConfigurationPath(base, item.Route))
		if err != nil || actual != item.Fingerprint {
			return false, err
		}
	}
	if len(retiredRouteCodes(item)) != 0 {
		actual, err := readFileFingerprint(ctx, target, files, routeConfigurationPath(base, item.Previous.Route))
		if err != nil || actual != "" {
			return false, err
		}
	}
	return true, nil
}

func (m *RouteManager) commitPublicationFiles(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication, body []byte) error {
	if item.Route.Enabled {
		active := routeConfigurationPath(base, item.Route)
		if err := m.runtime.SyncFiles(ctx, target, path.Dir(active), []deploymentport.WorkspaceFile{{Path: active, Content: body, Mode: 0o600}}, ""); err != nil {
			return err
		}
	}
	if len(retiredRouteCodes(item)) != 0 {
		return files.RemoveFile(ctx, target, routeConfigurationPath(base, item.Previous.Route))
	}
	return nil
}

func (m *RouteManager) restorePublicationFiles(ctx context.Context, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication, oldBody []byte) error {
	if err := validatePublicationFiles(ctx, target, files, base, item); err != nil {
		return err
	}
	previousEnabled := previousFileAvailable(item)
	if previousEnabled {
		if digest(oldBody) != item.Previous.Fingerprint {
			return errors.New("previous Route recovery file does not match its publication record")
		}
		active := routeConfigurationPath(base, item.Previous.Route)
		actual, err := readFileFingerprint(ctx, target, files, active)
		if err != nil {
			return err
		}
		if actual != item.Previous.Fingerprint {
			if err := m.runtime.SyncFiles(ctx, target, path.Dir(active), []deploymentport.WorkspaceFile{{Path: active, Content: oldBody, Mode: 0o600}}, ""); err != nil {
				return err
			}
		}
	}
	if item.Route.Enabled && (!previousEnabled || item.Route.Name != item.Previous.Route.Name) {
		if err := files.RemoveFile(ctx, target, routeConfigurationPath(base, item.Route)); err != nil {
			return err
		}
	}
	if !previousEnabled && item.Previous != nil && item.Previous.Route.Enabled && (!item.Route.Enabled || item.Route.Name != item.Previous.Route.Name) {
		return files.RemoveFile(ctx, target, routeConfigurationPath(base, item.Previous.Route))
	}
	return nil
}

func (m *RouteManager) rollbackPublication(ctx context.Context, projectId string, gateway model.GatewayConfig, target environmentport.Target, files deploymentport.WorkspaceFiles, base string, item routeport.Publication, oldBody []byte) error {
	if err := m.restorePublicationFiles(ctx, target, files, base, item, oldBody); err != nil {
		return err
	}
	var absent []string
	if item.Route.Enabled && (!previousFileAvailable(item) || item.Previous.Route.Name != item.Route.Name) {
		absent = []string{item.Route.Name}
	}
	if !previousFileAvailable(item) && item.Previous != nil && item.Previous.Route.Enabled && (!item.Route.Enabled || item.Route.Name != item.Previous.Route.Name) {
		absent = append(absent, item.Previous.Route.Name)
	}
	if err := m.loadFileConfiguration(ctx, projectId, gateway, target, oldBody, absent...); err != nil {
		return err
	}
	return m.restoreRecord(ctx, target, base, item.Previous, item, nil)
}
