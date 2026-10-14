package deploymentsvc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	mountSourceDirectory      = "directory"
	mountSourceFile           = "file"
	mountSourceNamedVolume    = "named_volume"
	mountSourceControlledFile = "controlled_file"

	maxMountContentBytes = 256 * 1024
)

type MountSpec = model.VersionComponentMount

type ResolvedMount struct {
	Relative          bool
	Source            string
	Target            string
	ReadOnly          bool
	HostSource        string
	LogicalSource     string
	IsFile            bool
	ShouldMaterialize bool
	SourceType        string
	Content           string
	IgnoreIfExists    bool
	FileMode          os.FileMode
	NamedVolumeName   string
}

func validateMountSpec(m MountSpec) error {
	if err := model.ValidateSharedMount(m.SourceType, m.Source, m.Shared); err != nil {
		return err
	}
	if m.SourceType == "" || m.Source == "" || m.Target == "" {
		return fmt.Errorf("source_type, source and target are required")
	}
	if m.SourceIsHostPath {
		if m.SourceType != mountSourceDirectory && m.SourceType != mountSourceFile {
			return fmt.Errorf("source_is_host_path is only allowed for directory or file mounts")
		}
		if !isAbsoluteMountSource(m.Source) {
			return fmt.Errorf("host path source must be an absolute path")
		}
	} else if m.SourceType == mountSourceDirectory || m.SourceType == mountSourceFile || m.SourceType == mountSourceControlledFile {
		if !isExplicitMountSource(m.Source) {
			return fmt.Errorf("source must be an absolute path or a relative path starting with ./ without parent directory segments")
		}
	}
	switch m.SourceType {
	case mountSourceDirectory, mountSourceFile:
		if m.Content != "" || m.Mode != "" || m.IgnoreIfExists {
			return fmt.Errorf("content options are only allowed on controlled_file mounts")
		}
	case mountSourceNamedVolume:
		if strings.ContainsAny(m.Source, `/\\`) {
			return fmt.Errorf("named_volume source must be a volume name")
		}
		if m.SourceIsHostPath || m.Content != "" || m.Mode != "" || m.IgnoreIfExists {
			return fmt.Errorf("named_volume does not support file source options")
		}
	case mountSourceControlledFile:
		if m.SourceIsHostPath {
			return fmt.Errorf("controlled_file source_is_host_path must be false")
		}
		if len(m.Content) > maxMountContentBytes {
			return fmt.Errorf("content exceeds %d bytes", maxMountContentBytes)
		}
		if _, err := parseUnixFileMode(m.Mode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported source_type %q", m.SourceType)
	}
	return nil
}

func isExplicitMountSource(source string) bool {
	if isAbsoluteMountSource(source) {
		return true
	}
	return strings.HasPrefix(source, "./") && !strings.Contains(source, "\\") && !hasParentMountSegment(source)
}

func parseUnixFileMode(mode string) (os.FileMode, error) {
	if len(mode) != 4 || mode[0] != '0' {
		return 0, fmt.Errorf("controlled_file mode must be a four-digit Unix octal mode")
	}
	parsed, err := strconv.ParseUint(mode, 8, 32)
	if err != nil || parsed > 0o777 {
		return 0, fmt.Errorf("controlled_file mode must be a four-digit Unix octal mode")
	}
	return os.FileMode(parsed), nil
}

func isAbsoluteMountSource(source string) bool {
	if source == "" {
		return false
	}
	if strings.HasPrefix(source, "/") {
		return true
	}
	if len(source) >= 3 && source[1] == ':' && source[2] == '/' {
		letter := source[0]
		if (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z') {
			return true
		}
	}
	return false
}

func hasParentMountSegment(source string) bool {
	for _, segment := range strings.Split(source, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func resolveMountSpecsForPaths(mounts []MountSpec, logicalServiceDir string, composeMountSourceDir string) ([]ResolvedMount, error) {
	out := make([]ResolvedMount, 0, len(mounts))
	for _, mount := range mounts {
		if err := validateMountSpec(mount); err != nil {
			return nil, err
		}
		item := ResolvedMount{Source: mount.Source, Target: mount.Target, ReadOnly: mount.ReadOnly, Relative: strings.HasPrefix(mount.Source, "./"), SourceType: mount.SourceType, Content: mount.Content, IgnoreIfExists: mount.IgnoreIfExists}
		if mount.SourceType == mountSourceControlledFile {
			mode, err := parseUnixFileMode(mount.Mode)
			if err != nil {
				return nil, err
			}
			item.FileMode = mode
		}
		switch mount.SourceType {
		case mountSourceDirectory, mountSourceFile, mountSourceControlledFile:
			if mount.SourceIsHostPath {
				item.HostSource = mount.Source
				item.IsFile = mount.SourceType == mountSourceFile
				item.Source = item.HostSource
				break
			}
			absoluteSource := isAbsoluteMountSource(mount.Source)
			if absoluteSource {
				item.LogicalSource = filepath.FromSlash(mount.Source)
			} else {
				if logicalServiceDir == "" {
					return nil, fmt.Errorf("logical service dir required for %s mount %s", mount.SourceType, mount.Source)
				}
				item.LogicalSource = filepath.Join(logicalServiceDir, filepath.FromSlash(mount.Source))
			}
			item.IsFile = mount.SourceType == mountSourceFile || mount.SourceType == mountSourceControlledFile
			item.ShouldMaterialize = mount.SourceType == mountSourceDirectory || mount.SourceType == mountSourceControlledFile
			if composeMountSourceDir != "" && absoluteSource {
				item.HostSource = mount.Source
				item.Source = item.HostSource
			} else if composeMountSourceDir != "" {
				item.HostSource = filepath.ToSlash(filepath.Join(composeMountSourceDir, filepath.FromSlash(mount.Source)))
				item.Source = item.HostSource
			}
		case mountSourceNamedVolume:
			item.NamedVolumeName = mount.Source
		}
		out = append(out, item)
	}
	return out, nil
}
