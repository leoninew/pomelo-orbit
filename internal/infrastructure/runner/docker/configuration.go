package dockerrunner

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// ConfigurationServices decodes only the service/image facts used for drift checks.
func (RuntimeInspector) ConfigurationServices(content string) (map[string]string, error) {
	var document struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil, err
	}
	if document.Services == nil {
		return nil, errors.New("compose configuration is missing services")
	}
	services := make(map[string]string, len(document.Services))
	for name, service := range document.Services {
		services[name] = service.Image
	}
	return services, nil
}
