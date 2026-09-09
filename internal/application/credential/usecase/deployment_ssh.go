package credentialsvc

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"strings"

	"golang.org/x/crypto/ssh"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

const defaultDeploymentSSHKeyName = "deployment-ssh"

func generateDeploymentSSHKeypair() (credentialdto.DeploymentSSHPrivateKey, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return credentialdto.DeploymentSSHPrivateKey{}, apperror.Wrap(apperror.KindInternal, "Failed to generate deployment SSH key", err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return credentialdto.DeploymentSSHPrivateKey{}, apperror.Wrap(apperror.KindInternal, "Failed to encode deployment SSH private key", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return credentialdto.DeploymentSSHPrivateKey{}, apperror.Wrap(apperror.KindInternal, "Failed to derive deployment SSH public key", err)
	}
	return credentialdto.DeploymentSSHPrivateKey{
		PrivateKey: string(pem.EncodeToMemory(block)),
		PublicKey:  strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))),
	}, nil
}

func encodeDeploymentSSHPayload(payload credentialdto.DeploymentSSHPrivateKey) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to encode deployment SSH credential", err)
	}
	return string(encoded), nil
}
