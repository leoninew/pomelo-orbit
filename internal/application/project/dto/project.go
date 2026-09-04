package dto

type EnvironmentCreateInput struct {
	State                      string
	Platform                   string
	Host                       string
	Port                       int
	Username                   string
	WorkspaceRoot              string
	DeploymentSSHKeyName       string
	DeploymentSSHPrivateKey    string
	DeploymentSSHKeyPassphrase string
	HostKeyFingerprint         string
}

type CreateInput struct {
	Name        string
	Code        string
	Environment EnvironmentCreateInput
}

type SaveInput struct {
	Name string
}
