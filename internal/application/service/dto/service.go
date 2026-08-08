package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

// ServiceView is the runtime binding for an application instance, with display labels.
type ServiceView struct {
	Service              model.Service
	Env                  []model.ServiceEnv
	Components           []model.ServiceComponent
	ComponentDefinitions []model.VersionComponent
	PendingDeploy        bool
	EffectivePlanHash    string
	EffectiveError       string
	ApplicationName      string
	ApplicationCode      string
	ApplicationKind      string
	VersionLabel         string
}

// ServiceListInput filters project-scoped service listing.
type ServiceListInput struct {
	ProjectId     string
	ApplicationId string
	Status        string
	Search        string
	Page          int
	PerPage       int
}

// ServiceTargetInput identifies a runtime binding for a command or runtime query.
type ServiceTargetInput struct {
	ServiceId string
}

type ServiceCreateInput struct {
	ApplicationId string
	VersionId     string
	InstanceKey   string
	Code          string
}

type ServiceComponentOverlayInput struct {
	Env       []model.ServiceComponentEnv
	Mounts    []model.ServiceComponentMount
	Resources *model.ServiceComponentResources
	Endpoints []model.ServiceComponentEndpoint
}

// ServiceComponentDetail compares a Version declaration and its sparse Service
// overlay. Effective is unavailable while the Service configuration is incomplete.
type ServiceComponentDetail struct {
	Component   model.ServiceComponent
	Declaration model.VersionComponent
	Effective   *model.EffectiveServiceComponent
}

type ServiceBasicUpdateInput struct {
	VersionId   string
	InstanceKey string
}

type ServiceEnvUpdateInput struct {
	Env []model.ServiceEnv
}
