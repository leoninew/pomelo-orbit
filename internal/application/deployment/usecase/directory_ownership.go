package deploymentsvc

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/common/workspacepath"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s ExecutionService) checkDirectoryOwnership(ctx context.Context, target environmentport.Target, service model.Service, directory string, mounts []ResolvedMount) error {
	services, err := s.executionStore.DirectoryServices(ctx, service.ProjectId)
	if err != nil {
		return err
	}
	platform := runtime.GOOS
	if target.Environment.IsSSH() {
		platform = target.Environment.SSH.Platform
	}
	physical, err := s.runtime.ComposeMountSourceDir(ctx, target, deploymentport.ServiceLocation{Code: service.Code, Directory: directory})
	if err != nil {
		return err
	}
	physicalSources := []string{physical}
	for _, mount := range mounts {
		if mount.Relative {
			physicalSources = append(physicalSources, mount.HostSource)
		}
	}
	for _, other := range services {
		if other.Id == service.Id {
			continue
		}
		candidates := []struct {
			directory string
			revision  int64
		}{{other.DeploymentDirectory, other.DirectoryTargetRevision}, {other.RuntimeDirectory, other.RuntimeTargetRevision}}
		for _, candidate := range candidates {
			if candidate.directory == "" || candidate.revision != target.Environment.TargetRevision {
				continue
			}
			location := deploymentport.ServiceLocation{Code: other.Code, Directory: candidate.directory}
			logical, err := s.runtime.ResolveDirectory(ctx, target, location)
			if err != nil {
				return fmt.Errorf("resolve service %s directory: %w", other.Code, err)
			}
			if workspacepath.OverlappingDirectories(directory, logical, platform) {
				return fmt.Errorf("deployment directory overlaps service %s", other.Code)
			}
			mapped, err := s.runtime.ComposeMountSourceDir(ctx, target, location)
			if err != nil {
				return err
			}
			for _, source := range physicalSources {
				if workspacepath.OverlappingDirectories(source, mapped, platform) {
					return fmt.Errorf("docker mount directory overlaps service %s", other.Code)
				}
			}
		}
		if err := s.checkRunningMountOwnership(ctx, target, other.Code, physicalSources, platform); err != nil {
			return err
		}
	}
	return nil
}

func (s ExecutionService) checkRunningMountOwnership(ctx context.Context, target environmentport.Target, serviceCode string, sources []string, platform string) error {
	mounts, err := s.mountInventory.RunningBindMounts(ctx, target, serviceCode)
	if err != nil {
		return err
	}
	for _, mount := range mounts {
		occupied := dockerMountComparisonPath(mount.Source, platform)
		for _, source := range sources {
			if mount.Shared && mountSourceContains(occupied, source, platform) {
				continue
			}
			if workspacepath.OverlappingDirectories(dockerMountComparisonPath(source, platform), occupied, platform) {
				return fmt.Errorf("docker mount directory %s overlaps bind mount %s used by running service %s", source, mount.Source, serviceCode)
			}
		}
	}
	return nil
}
func dockerMountComparisonPath(value, platform string) string {
	if platform == model.EnvironmentPlatformWindows || len(value) > 1 && value[1] == ':' {
		value = strings.ReplaceAll(value, "\\", "/")
		value = strings.ToLower(value)
	}
	// Docker Desktop may report a Windows bind source using its Linux VM path.
	for _, prefix := range []string{"/run/desktop/mnt/host/", "/host_mnt/"} {
		if strings.HasPrefix(value, prefix) {
			suffix := strings.TrimPrefix(value, prefix)
			if len(suffix) >= 2 && suffix[1] == '/' && (suffix[0] >= 'a' && suffix[0] <= 'z' || suffix[0] >= 'A' && suffix[0] <= 'Z') {
				return strings.ToLower(suffix[:1] + ":" + suffix[1:])
			}
		}
	}
	return value
}
