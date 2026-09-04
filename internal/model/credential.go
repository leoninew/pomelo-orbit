package model

import "time"

const (
	CredentialTypeDeploymentSSHPrivateKey = "deployment_ssh_private_key"

	// DeploymentSSHCredentialReconfigurationPlaceholder is the exact migration marker
	// for a legacy Project that must receive a new SSH key before it can deploy.
	DeploymentSSHCredentialReconfigurationPlaceholder = "__POMELO_ORBIT_DEPLOYMENT_SSH_RECONFIGURATION_REQUIRED__"
)

type Credential struct {
	Id            string    `db:"id"`
	ProjectId     *string   `db:"project_id"`
	Name          string    `db:"name"`
	Type          string    `db:"type"`
	EncryptedData string    `db:"encrypted_data"`
	Revision      int64     `db:"revision"`
	CreatedAt     time.Time `db:"created_at"`
}

func (c Credential) IsDeploymentSSHPrivateKey() bool {
	return c.Type == CredentialTypeDeploymentSSHPrivateKey
}

func (c Credential) RequiresDeploymentSSHCredentialReconfiguration() bool {
	return c.IsDeploymentSSHPrivateKey() && c.EncryptedData == DeploymentSSHCredentialReconfigurationPlaceholder
}
