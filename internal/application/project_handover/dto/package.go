package dto

import (
	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
)

const (
	Format        = "pomelo-orbit/project-handover"
	FormatVersion = 1
)

type Package struct {
	Format       string                                 `json:"format"`
	Version      int                                    `json:"version"`
	Project      projectdto.ProjectDefinition           `json:"project"`
	Environment  environmentdto.TargetDefinition        `json:"environment"`
	Applications []applicationdto.ApplicationDefinition `json:"applications"`
	Gateway      *gatewaydto.GatewayDefinition          `json:"gateway,omitempty"`
	Services     []servicedto.ServiceDefinition         `json:"services"`
	Routes       []routedto.RouteDefinitionInput        `json:"routes"`
}

type ImportMode string

const (
	ImportModeNew     ImportMode = "new"
	ImportModeReplace ImportMode = "replace"
)

type ImportInput struct {
	Mode            ImportMode
	Name            string
	Code            string
	TargetProjectID string
	Package         Package
}
