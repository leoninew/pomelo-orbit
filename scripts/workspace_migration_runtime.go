//go:build ignore

// The Python migration tool builds this helper automatically. It reuses Orbit's
// config, credential crypto and target runners without starting the application.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	"github.com/leoninew/pomelo-orbit/internal/config"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type connection struct {
	Environment struct {
		Id                 string `json:"id"`
		ProjectId          string `json:"project_id"`
		TargetType         string `json:"target_type"`
		WorkspaceRoot      string `json:"workspace_root"`
		TargetRevision     int64  `json:"target_revision"`
		Platform           string `json:"platform"`
		Host               string `json:"host"`
		Port               int    `json:"port"`
		Username           string `json:"username"`
		CredentialId       string `json:"ssh_credential_id"`
		CredentialRevision int64  `json:"ssh_credential_revision"`
		Fingerprint        string `json:"host_key_fingerprint"`
	} `json:"environment"`
	Credential struct {
		Id                  string `json:"id"`
		EnvironmentId       string `json:"environment_id"`
		Revision            int64  `json:"revision"`
		PublicKey           string `json:"public_key"`
		EncryptedPrivateKey string `json:"encrypted_private_key"`
	} `json:"credential"`
}

func (c connection) target(secret string) (environmentport.Target, error) {
	e := c.Environment
	target := environmentport.Target{Environment: model.Environment{Id: e.Id, ProjectId: e.ProjectId,
		TargetType: e.TargetType, WorkspaceRoot: e.WorkspaceRoot, TargetRevision: e.TargetRevision}}
	if e.TargetType == model.EnvironmentTargetTypeLocal {
		return target, nil
	}
	if e.TargetType != model.EnvironmentTargetTypeSSH || c.Credential.Id != e.CredentialId ||
		c.Credential.EnvironmentId != e.Id || c.Credential.Revision != e.CredentialRevision {
		return target, errors.New("saved SSH credential does not match the environment binding")
	}
	key, err := security.DecryptString(secret, c.Credential.EncryptedPrivateKey)
	if err != nil {
		return target, errors.New("cannot decrypt existing environment SSH credential")
	}
	target.Environment.SSH = &model.EnvironmentSSHTarget{Platform: e.Platform, Host: e.Host, Port: e.Port,
		Username: e.Username, CredentialId: e.CredentialId, CredentialRevision: e.CredentialRevision, HostKeyFingerprint: e.Fingerprint}
	target.PrivateKey = &environmentdto.DeploymentSSHPrivateKey{PrivateKey: key, PublicKey: c.Credential.PublicKey}
	return target, nil
}

func execute() (any, error) {
	var input struct {
		Action     string     `json:"action"`
		Connection connection `json:"connection"`
		Argv       []string   `json:"argv"`
		Workdir    string     `json:"workdir"`
		Stdin      []byte     `json:"stdin"`
		Timeout    int        `json:"timeout"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return nil, errors.New("invalid helper request")
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.New("existing Orbit startup configuration could not be loaded")
	}
	if input.Action == "config" {
		return map[string]any{"profile": cfg.App.Env, "database_url": cfg.Database.Url,
			"workspace_root": cfg.Workspace.Root, "deployment_root": cfg.Logging.DeploymentRoot}, nil
	}
	if input.Timeout <= 0 {
		input.Timeout = 120
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(input.Timeout)*time.Second)
	defer cancel()
	target, err := input.Connection.target(cfg.Jwt.SecretKey)
	if err != nil {
		return nil, err
	}
	switch input.Action {
	case "exec":
		if !target.Environment.IsSSH() {
			return nil, errors.New("remote exec requires an SSH environment")
		}
		session, openErr := sshrunner.NewCommandRunner().Open(ctx, target)
		if openErr != nil {
			return nil, errors.New("SSH connection, host key verification or managed-key authentication failed")
		}
		defer func() { _ = session.Close() }()
		var stdout, stderr bytes.Buffer
		outcome, runErr := session.Run(ctx, environmentport.Command{Argv: input.Argv, Workdir: input.Workdir, Stdin: input.Stdin}, &stdout, &stderr)
		if runErr != nil || outcome.ExitCode == nil {
			return nil, errors.New("SSH command result is unknown; resume to inspect saved migration progress")
		}
		return map[string]any{"exit_code": *outcome.ExitCode, "stdout": stdout.String(), "stderr": stderr.String(), "fingerprint": session.HostKeyFingerprint()}, nil
	default:
		return nil, errors.New("unknown migration helper action")
	}
}

func main() {
	result, err := execute()
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"error": err.Error()})
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "cannot encode migration helper result")
		os.Exit(1)
	}
}
