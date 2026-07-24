package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

// ServiceView is the runtime binding for an application instance, with display labels.
type ServiceView struct {
	Service                    model.Service
	ApplicationName            string
	ApplicationCode            string
	ApplicationKind            string
	EnvironmentName            string
	EnvironmentCode            string
	VersionLabel               string
	LastSuccessfulVersionLabel *string
}

// ServiceListInput filters project-scoped service listing.
type ServiceListInput struct {
	ProjectId     string
	ApplicationId string
	EnvironmentId string
	Status        string
	Search        string
	Page          int
	PerPage       int
}

// ServiceTargetInput identifies a runtime binding for a command or runtime query.
type ServiceTargetInput struct {
	EnvironmentId string
	InstanceKey   string
	ServiceId     string
}
