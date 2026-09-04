package dto

type BootstrapInput struct {
	State              string
	Platform           string
	Host               string
	Port               int
	Username           string
	WorkspaceRoot      string
	HostKeyFingerprint string
}

type UpdateInput struct {
	State                      *string
	Platform                   *string
	Host                       *string
	Port                       *int
	Username                   *string
	WorkspaceRoot              *string
	DeploymentSSHPrivateKey    *string
	DeploymentSSHKeyPassphrase *string
	HostKeyFingerprint         *string
}
