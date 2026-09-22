package environmentsvc

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"golang.org/x/crypto/ssh"
)

func generateSSHKeypair() (publicKey string, privateKey string, err error) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", apperror.Wrap(apperror.KindInternal, "Failed to generate deployment SSH key", err)
	}
	block, err := ssh.MarshalPrivateKey(key, "")
	if err != nil {
		return "", "", apperror.Wrap(apperror.KindInternal, "Failed to encode deployment SSH private key", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return "", "", apperror.Wrap(apperror.KindInternal, "Failed to derive deployment SSH public key", err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))), string(pem.EncodeToMemory(block)), nil
}

func (s Service) createEnvironmentCredential(ctx context.Context, projectId string) (model.EnvironmentCredential, error) {
	if s.environmentCredentials == nil {
		return model.EnvironmentCredential{}, apperror.New(apperror.KindInternal, "environment credential store is not configured")
	}
	if strings.TrimSpace(s.secretKey) == "" {
		return model.EnvironmentCredential{}, apperror.New(apperror.KindInternal, "credential encryption key is not configured")
	}
	publicKey, privateKey, err := generateSSHKeypair()
	if err != nil {
		return model.EnvironmentCredential{}, err
	}
	encrypted, err := security.EncryptString(s.secretKey, privateKey)
	if err != nil {
		return model.EnvironmentCredential{}, apperror.Wrap(apperror.KindInternal, "Failed to encrypt deployment SSH private key", err)
	}
	item := model.EnvironmentCredential{
		Id:                  idutil.NewId(),
		ProjectId:           projectId,
		PublicKey:           publicKey,
		EncryptedPrivateKey: encrypted,
		Revision:            1,
		CreatedAt:           time.Now().UTC(),
	}
	if err := s.environmentCredentials.CreateEnvironmentCredential(ctx, item); err != nil {
		return model.EnvironmentCredential{}, apperror.Wrap(apperror.KindInternal, "Failed to create environment credential", err)
	}
	return item, nil
}

func (s Service) environmentCredential(ctx context.Context, id string) (model.EnvironmentCredential, error) {
	if s.environmentCredentials == nil {
		return model.EnvironmentCredential{}, apperror.New(apperror.KindInternal, "environment credential store is not configured")
	}
	item, err := s.environmentCredentials.EnvironmentCredential(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.EnvironmentCredential{}, apperror.New(apperror.KindNotFound, "Environment credential not found")
		}
		return model.EnvironmentCredential{}, apperror.Wrap(apperror.KindInternal, "Failed to load environment credential", err)
	}
	return item, nil
}

// ensureEnvironmentCredential only serves the explicit SSH initialization
// command. It never selects an unbound credential from the Project history.
func (s Service) ensureEnvironmentCredential(ctx context.Context, environment model.Environment) (model.EnvironmentCredential, error) {
	if hasSSHCredentialBinding(environment) {
		credential, err := s.environmentCredential(ctx, environment.SSH.CredentialId)
		if err == nil && matchesEnvironmentCredential(environment, credential) {
			return credential, nil
		}
		if err != nil && !apperror.IsKind(err, apperror.KindNotFound) {
			return model.EnvironmentCredential{}, err
		}
	}
	return s.createEnvironmentCredential(ctx, environment.ProjectId)
}

func decryptEnvironmentPrivateKey(secretKey string, item model.EnvironmentCredential) (environmentdto.DeploymentSSHPrivateKey, error) {
	privateKey, err := security.DecryptString(secretKey, item.EncryptedPrivateKey)
	if err != nil {
		return environmentdto.DeploymentSSHPrivateKey{}, apperror.Wrap(apperror.KindInternal, "Failed to decrypt deployment SSH private key", err)
	}
	return environmentdto.DeploymentSSHPrivateKey{
		PrivateKey: privateKey,
		PublicKey:  item.PublicKey,
	}, nil
}

func matchesEnvironmentCredential(environment model.Environment, credential model.EnvironmentCredential) bool {
	return environment.IsSSH() &&
		credential.Id == environment.SSH.CredentialId &&
		credential.ProjectId == environment.ProjectId &&
		credential.Revision == environment.SSH.CredentialRevision
}
