package dto

import "github.com/leoninew/pomelo-orbit/internal/model"

// ServiceView is the runtime binding for an application instance, with display labels.
type ServiceView struct {
	Service              model.Service
	Env                  []model.ServiceEnv
	Components           []model.ServiceComponent
	ComponentDefinitions []model.VersionComponent
	EffectiveComponents  []model.EffectiveServiceComponent
	PendingDeploy        bool
	ActiveDeployment     bool
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
	Entrypoint    *string
	Command       *string
	PullPolicy    *string
	RestartPolicy *string
	Env           []model.ServiceComponentEnv
	Mounts        []model.ServiceComponentMount
	Resources     *model.ServiceComponentResources
	Endpoints     []model.ServiceComponentEndpoint
}

// ServiceComponentDetail keeps the Version declaration and sparse Service
// Component values separate. An absent Service value means inheritance.
type ServiceComponentDetail struct {
	ServiceComponent model.ServiceComponent
	VersionComponent model.VersionComponent
}

type ServiceBasicUpdateInput struct {
	VersionId   string
	InstanceKey string
}

type ServiceEnvUpdateInput struct {
	Env []model.ServiceEnv
}
