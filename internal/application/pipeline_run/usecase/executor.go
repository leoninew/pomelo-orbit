package pipelinerunsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	pipelinerunport "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/port"
	repositoryport "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type Executor struct {
	store            pipelineExecutionStore
	workspace        pipelinerunport.Workspace
	logStore         pipelinerunport.ExecutionLogStore
	secretKey        string
	logger           *slog.Logger
	executionTimeout time.Duration
	runner           pipelinerunport.ContainerRunner
	localSource      repositoryport.LocalDirectorySource
}

func (e Executor) Execute(ctx context.Context, executionCtx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, stages []model.StageDefinition) (bool, string) {
	layers, err := topologicalLayers(stages)
	if err != nil {
		e.logger.Error("cyclic dependency", "run", run.Id, "error", err)
		return false, fmt.Sprintf("Cyclic dependency detected: %v", err)
	}
	stageById := map[string]model.StageDefinition{}
	for _, stage := range stages {
		stageById[stage.Id] = stage
	}

	for layerIndex, layer := range layers {
		e.logger.Info("executing layer", "index", layerIndex+1, "total", len(layers), "run", run.Id, "stages", layer)
		results := e.executeLayer(ctx, executionCtx, run, repo, variables, layer, stageById)
		failed := make([]string, 0)
		for _, stageId := range layer {
			stage := stageById[stageId]
			if !results[stage.Name] {
				failed = append(failed, stage.Name)
			}
		}
		if len(failed) > 0 {
			e.cancelRemaining(ctx, run.Id, layers, layerIndex+1, stageById)
			return false, fmt.Sprintf("Stage(s) failed: %s", strings.Join(failed, ", "))
		}
	}
	return true, ""
}

func (e Executor) executeLayer(ctx context.Context, executionCtx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, layer []string, stages map[string]model.StageDefinition) map[string]bool {
	results := make(map[string]bool, len(layer))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, stageId := range layer {
		stage := stages[stageId]
		wg.Go(func() {
			ok := e.executeStage(ctx, executionCtx, run, repo, variables, stage)
			mu.Lock()
			results[stage.Name] = ok
			mu.Unlock()
		})
	}
	wg.Wait()
	return results
}

func (e Executor) executeStage(ctx context.Context, executionCtx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, stage model.StageDefinition) bool {
	pipelineStageRun := model.PipelineStageRun{Id: idutil.NewId(), PipelineRunId: run.Id, StageId: stage.Id, StageName: stage.Name, Status: status.WorkStatusWaitingToRun}
	if err := e.store.InsertPipelineStageRun(ctx, pipelineStageRun); err != nil {
		e.logger.Error("pipeline stage run insert failed", "run", run.Id, "stage", stage.Name, "error", err)
		return false
	}

	started := now()
	pipelineStageRun.StartedAt = &started
	pipelineStageRun.Status = status.WorkStatusRunning
	_ = e.store.UpdatePipelineStageRun(ctx, pipelineStageRun)
	if err := executionCtx.Err(); err != nil {
		return e.failStage(ctx, pipelineStageRun, e.executionErrorMessage(err))
	}

	logPath := e.workspace.StageLogPath(run.Id, pipelineStageRun.Id)
	logWriter, err := e.logStore.Writer(logPath)
	if err != nil {
		return e.failStage(ctx, pipelineStageRun, fmt.Sprintf("create log writer: %v", err))
	}
	defer func() { _ = logWriter.Close() }()

	volumes, err := e.workspace.DockerStageMounts(ctx, repo.Code, run.Id)
	if err != nil {
		return e.failStage(ctx, pipelineStageRun, err.Error())
	}
	if repo.RepositoryType == model.RepositoryTypeLocalDirectory {
		if e.localSource == nil {
			return e.failStage(ctx, pipelineStageRun, "local directory sources are disabled")
		}
		sourcePath, err := e.localSource.DockerHostPath(ctx, repo.RepositoryUrl)
		if err != nil {
			return e.failStage(ctx, pipelineStageRun, err.Error())
		}
		volumes = append(volumes, pipelinerunport.VolumeMount{HostPath: sourcePath, ContainerPath: "/source", Mode: "ro"})
	}

	script, environment, err := e.pipelineStageRunConfig(ctx, repo, variables, stage)
	if err != nil {
		return e.failStage(ctx, pipelineStageRun, err.Error())
	}

	exitCode, _, err := e.runner.Run(executionCtx, pipelinerunport.RunOptions{
		ContainerName: "pomelo-orbit-stage-" + pipelineStageRun.Id,
		Image:         stage.Image,
		Script:        safeCommand(script),
		Environment:   environment,
		Volumes:       volumes,
		LogWriter:     logWriter,
	})
	if err != nil {
		if executionErr := executionCtx.Err(); executionErr != nil {
			message := e.executionErrorMessage(executionErr)
			if err != executionErr {
				message = fmt.Sprintf("%s: %v", message, err)
			}
			return e.failStage(ctx, pipelineStageRun, message)
		}
		return e.failStage(ctx, pipelineStageRun, err.Error())
	}
	if err := executionCtx.Err(); err != nil {
		return e.failStage(ctx, pipelineStageRun, e.executionErrorMessage(err))
	}
	if exitCode != 0 {
		if err := logWriter.Close(); err != nil {
			return e.failStage(ctx, pipelineStageRun, fmt.Sprintf("close stage log writer: %v", err))
		}
		content, _, _ := e.logStore.Read(logPath, 0)
		errMsg := lastLines(string(content), 3)
		return e.completeStageFailed(ctx, pipelineStageRun, exitCode, errMsg)
	}

	finished := now()
	pipelineStageRun.FinishedAt = &finished
	pipelineStageRun.Status = status.WorkStatusRanToCompletion
	pipelineStageRun.ExitCode = &exitCode
	if err := e.saveArtifacts(ctx, run, stage); err != nil {
		return e.failStage(ctx, pipelineStageRun, err.Error())
	}
	if err := e.store.UpdatePipelineStageRun(ctx, pipelineStageRun); err != nil {
		e.logger.Error("pipeline stage run update failed", "run", run.Id, "stage", stage.Name, "error", err)
		return false
	}
	e.logger.Info("stage succeeded", "run", run.Id, "stage", stage.Name)
	return true
}

func (e Executor) executionErrorMessage(err error) string {
	if err == context.DeadlineExceeded {
		return fmt.Sprintf("pipeline execution timed out after %s", e.executionTimeout)
	}
	return fmt.Sprintf("pipeline execution canceled: %v", err)
}

func (e Executor) pipelineStageRunConfig(ctx context.Context, repo model.Repository, variables map[string]any, stage model.StageDefinition) (string, []string, error) {
	script := commandLines(stage.Script)
	environment := envMap(variables)
	if repo.RepositoryType == model.RepositoryTypeLocalDirectory || repo.GitCredentialId == nil || !stageUsesRepositoryUrl(stage.Script, repo.RepositoryUrl) {
		return script, environment, nil
	}
	credential, err := e.store.Credential(ctx, *repo.GitCredentialId)
	if err != nil {
		return "", nil, fmt.Errorf("load git credential: %w", err)
	}
	authenticatedUrl, err := e.authenticatedRepositoryUrl(repo.RepositoryUrl, credential)
	if err != nil {
		return "", nil, err
	}
	environment = append(environment, gitCredentialEnvironment(repo.RepositoryUrl, authenticatedUrl)...)
	return script, environment, nil
}

func stageUsesRepositoryUrl(script string, repositoryUrl string) bool {
	return strings.TrimSpace(repositoryUrl) != "" && strings.Contains(script, repositoryUrl)
}

func (e Executor) authenticatedRepositoryUrl(repositoryUrl string, credential model.Credential) (string, error) {
	credentialData, err := security.DecryptString(e.secretKey, credential.EncryptedData)
	if err != nil {
		return "", fmt.Errorf("decrypt git credential: %w", err)
	}
	switch credential.Type {
	case "github_token":
		return buildAuthenticatedRepositoryUrl(repositoryUrl, credentialData)
	case "gitee_token":
		username, token, err := splitGiteeCredential(credentialData)
		if err != nil {
			return "", err
		}
		return buildAuthenticatedRepositoryUrl(repositoryUrl, username+":"+token)
	case "git_ssh":
		return "", fmt.Errorf("git_ssh credentials are not supported by pomelo-orbit pipeline execution")
	default:
		return "", fmt.Errorf("unsupported git credential type: %s", credential.Type)
	}
}

func splitGiteeCredential(value string) (string, string, error) {
	username, token, ok := strings.Cut(value, ":")
	username = strings.TrimSpace(username)
	if !ok || username == "" || token == "" {
		return "", "", fmt.Errorf("gitee_token credential must be username:token")
	}
	return username, token, nil
}

func buildAuthenticatedRepositoryUrl(repositoryUrl string, authPart string) (string, error) {
	if strings.TrimSpace(authPart) == "" {
		return "", fmt.Errorf("git credential data is empty")
	}
	parsed, err := url.Parse(repositoryUrl)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("unsupported repository URL format: %s", repositoryUrl)
	}
	if strings.Contains(authPart, ":") {
		username, password, _ := strings.Cut(authPart, ":")
		parsed.User = url.UserPassword(username, password)
	} else {
		parsed.User = url.User(authPart)
	}
	return parsed.String(), nil
}

func gitCredentialEnvironment(repositoryUrl string, authenticatedUrl string) []string {
	return []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=url." + authenticatedUrl + ".insteadOf",
		"GIT_CONFIG_VALUE_0=" + repositoryUrl,
	}
}

func (e Executor) completeStageFailed(ctx context.Context, pipelineStageRun model.PipelineStageRun, exitCode int, message string) bool {
	finished := now()
	pipelineStageRun.FinishedAt = &finished
	pipelineStageRun.Status = status.WorkStatusFaulted
	pipelineStageRun.ExitCode = &exitCode
	pipelineStageRun.ErrorMessage = &message
	_ = e.store.UpdatePipelineStageRun(ctx, pipelineStageRun)
	e.logger.Warn("stage failed", "run", pipelineStageRun.PipelineRunId, "stage", pipelineStageRun.StageName, "exit_code", exitCode)
	return false
}

func (e Executor) failStage(ctx context.Context, pipelineStageRun model.PipelineStageRun, message string) bool {
	finished := now()
	pipelineStageRun.FinishedAt = &finished
	pipelineStageRun.Status = status.WorkStatusFaulted
	pipelineStageRun.ErrorMessage = &message
	_ = e.store.UpdatePipelineStageRun(ctx, pipelineStageRun)
	e.logger.Error("stage faulted", "run", pipelineStageRun.PipelineRunId, "stage", pipelineStageRun.StageName, "error", message)
	return false
}

func (e Executor) cancelRemaining(ctx context.Context, runId string, layers [][]string, start int, stages map[string]model.StageDefinition) {
	for i := start; i < len(layers); i++ {
		for _, stageId := range layers[i] {
			stage := stages[stageId]
			_ = e.store.InsertPipelineStageRun(ctx, model.PipelineStageRun{Id: idutil.NewId(), PipelineRunId: runId, StageId: stage.Id, StageName: stage.Name, Status: status.WorkStatusCanceled})
		}
	}
}

func (e Executor) saveArtifacts(ctx context.Context, run model.PipelineRun, stage model.StageDefinition) error {
	for _, artifact := range stage.Artifacts {
		artifactPath := artifact.Path
		if artifact.Type == "binary" {
			exists, err := e.workspace.ArtifactExists(run.Id, artifact.Path)
			if err != nil {
				return fmt.Errorf("check artifact %s: %w", artifact.Path, err)
			}
			artifactPath = filepath.Join(e.workspace.ArtifactsPath(run.Id), artifact.Path)
			if !exists {
				e.logger.Warn("artifact file not found, skipping", "run", run.Id, "stage", stage.Name, "path", artifactPath)
				continue
			}
		}
		if err := e.store.InsertArtifact(ctx, run.ProjectId, run, stage.Name, artifact, artifactPath); err != nil {
			return err
		}
	}
	return nil
}

func topologicalLayers(stages []model.StageDefinition) ([][]string, error) {
	stageIds := map[string]bool{}
	inDegree := map[string]int{}
	children := map[string][]string{}
	for _, stage := range stages {
		stageIds[stage.Id] = true
		inDegree[stage.Id] = 0
	}
	for _, stage := range stages {
		for _, dep := range stage.DependsOn {
			if !stageIds[dep] {
				continue
			}
			inDegree[stage.Id]++
			children[dep] = append(children[dep], stage.Id)
		}
	}
	var layers [][]string
	processed := 0
	for processed < len(stages) {
		var layer []string
		for id, degree := range inDegree {
			if degree == 0 {
				layer = append(layer, id)
			}
		}
		if len(layer) == 0 {
			return nil, fmt.Errorf("pipeline stages contain a cycle")
		}
		sort.Strings(layer)
		layers = append(layers, layer)
		for _, id := range layer {
			delete(inDegree, id)
			processed++
			for _, child := range children[id] {
				inDegree[child]--
			}
		}
	}
	return layers, nil
}

func lastLines(output string, count int) string {
	parts := strings.Split(strings.TrimSpace(output), "\n")
	lines := make([]string, 0, count)
	for _, line := range parts {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}
	return strings.Join(lines, "\n")
}

func now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func safeCommand(script string) string {
	return strings.ReplaceAll(script, "\r\n", "\n")
}
