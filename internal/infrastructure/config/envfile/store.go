package envfile

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

type Store struct {
	path string
}

func NewStore(cfg config.Config) Store {
	return Store{path: envPath(cfg)}
}

func (s Store) Load(context.Context) (map[string]string, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	result := map[string]string{}
	for line := range strings.SplitSeq(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = strings.Trim(value, `"'`)
	}
	return result, nil
}

func (s Store) Set(ctx context.Context, values map[string]string) error {
	envMap, err := s.Load(ctx)
	if err != nil {
		return err
	}
	maps.Copy(envMap, values)
	return s.write(envMap)
}

func (s Store) Delete(ctx context.Context, keys []string) error {
	envMap, err := s.Load(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		delete(envMap, key)
	}
	return s.write(envMap)
}

func (s Store) write(envMap map[string]string) error {
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", key, envMap[key]))
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(s.path, []byte(content), 0o644)
}

func envPath(cfg config.Config) string {
	if cfg.EnvFilePath != "" {
		return filepath.Clean(cfg.EnvFilePath)
	}
	return filepath.Join(cfg.OrbitRoot(), ".env")
}
