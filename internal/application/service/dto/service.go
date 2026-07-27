package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

// ServiceView is the runtime binding for an application instance, with display labels.
type ServiceView struct {
	Service                    model.Service
	ApplicationName            string
	ApplicationCode            string
	ApplicationKind            string
	VersionLabel               string
	LastSuccessfulVersionLabel *string
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
	InstanceKey string
	ServiceId   string
}

// RuntimeEnvView is the current persisted runtime_env configuration for a service version.
// It describes the values that a subsequent deployment will materialize, not a container snapshot.
type RuntimeEnvView struct {
	ServiceId string
	VersionId string
	Items     []RuntimeEnvItem
}

type RuntimeEnvItem struct {
	ComponentName  string
	EnvKey         string
	CredentialId   string
	CredentialName string
	DataKey        string
	Value          string
}
