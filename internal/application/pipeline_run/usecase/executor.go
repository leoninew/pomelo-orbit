package pipelinerunsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	applicationport "github.com/leoninew/pomelo-orbit/internal/application/application/port"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	repositoryport "github.com/leoninew/pomelo-orbit/internal/application/repository/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var gitObjectIdPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

type Executor struct {
	store             pipelineExecutionStore
	versionForker     applicationport.BuildVersionForker
	transactionRunner pipelinerunport.TransactionRunner
	workspace         pipelinerunport.Workspace
	logStore          pipelinerunport.ExecutionLogStore
	secretKey         string
	logger            *slog.Logger
	executionTimeout  time.Duration
	runner            pipelinerunport.ContainerRunner
	localSource       repositoryport.LocalDirectorySource
}

func (e Executor) Execute(ctx context.Context, executionCtx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, stages []model.StageDefinition, stageRuns map[string]model.PipelineStageRun) (bool, string) {
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
		results := e.executeLayer(ctx, executionCtx, run, repo, variables, layer, stageById, stageRuns)
		failed := make([]string, 0)
		for _, stageId := range layer {
			stage := stageById[stageId]
			if !results[stage.Name] {
				failed = append(failed, stage.Name)
			}
		}
		if len(failed) > 0 {
			return false, fmt.Sprintf("Stage(s) failed: %s", strings.Join(failed, ", "))
		}
	}
	if err := e.forkBuildVersion(ctx, run, stages, fmt.Sprint(variables["runtime_datetime"])); err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (e Executor) executeLayer(ctx context.Context, executionCtx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, layer []string, stages map[string]model.StageDefinition, stageRuns map[string]model.PipelineStageRun) map[string]bool {
	results := make(map[string]bool, len(layer))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, stageId := range layer {
		stage := stages[stageId]
		wg.Go(func() {
			ok := e.executeStage(ctx, executionCtx, run, repo, variables, stage, stages, stageRuns[stage.Id])
			mu.Lock()
			results[stage.Name] = ok
			mu.Unlock()
		})
	}
	wg.Wait()
	return results
}

func (e Executor) executeStage(ctx context.Context, executionCtx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, stage model.StageDefinition, stages map[string]model.StageDefinition, pipelineStageRun model.PipelineStageRun) bool {
	if pipelineStageRun.Id == "" {
		e.logger.Error("pipeline stage run is missing", "run", run.Id, "stage", stage.Name)
		return false
	}
	begun, err := e.store.BeginPipelineStageRun(ctx, pipelineStageRun.Id)
	if err != nil {
		e.logger.Error("pipeline stage run begin failed", "run", run.Id, "stage", stage.Name, "error", err)
		return false
	}
	if !begun {
		return false
	}
	started := now()
	pipelineStageRun.StartedAt = &started
	pipelineStageRun.Status = status.WorkStatusRunning
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

	runOptions := pipelinerunport.RunOptions{
		ContainerName: "pomelo-orbit-stage-" + pipelineStageRun.Id,
		Image:         stage.Image,
		Script:        safeCommand(script),
		Environment:   environment,
		Volumes:       volumes,
		LogWriter:     logWriter,
	}
	exitCode, _, err := e.runner.Run(executionCtx, runOptions)
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
	runtimeDatetime, _ := variables["runtime_datetime"].(string)
	if err := e.saveArtifacts(executionCtx, run, stage, stages, runOptions, runtimeDatetime); err != nil {
		return e.failStage(ctx, pipelineStageRun, err.Error())
	}
	if _, err := e.store.CompletePipelineStageRun(ctx, pipelineStageRun); err != nil {
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
	_, _ = e.store.CompletePipelineStageRun(ctx, pipelineStageRun)
	e.logger.Warn("stage failed", "run", pipelineStageRun.PipelineRunId, "stage", pipelineStageRun.StageName, "exit_code", exitCode)
	return false
}

func (e Executor) failStage(ctx context.Context, pipelineStageRun model.PipelineStageRun, message string) bool {
	finished := now()
	pipelineStageRun.FinishedAt = &finished
	pipelineStageRun.Status = status.WorkStatusFaulted
	pipelineStageRun.ErrorMessage = &message
	_, _ = e.store.CompletePipelineStageRun(ctx, pipelineStageRun)
	e.logger.Error("stage faulted", "run", pipelineStageRun.PipelineRunId, "stage", pipelineStageRun.StageName, "error", message)
	return false
}

func (e Executor) saveArtifacts(ctx context.Context, run model.PipelineRun, stage model.StageDefinition, stages map[string]model.StageDefinition, runOptions pipelinerunport.RunOptions, runtimeDatetime string) error {
	for _, artifact := range stage.Artifacts {
		switch artifact.Collector {
		case "file":
			exists, err := e.workspace.ArtifactExists(run.Id, artifact.Reference)
			if err != nil {
				return fmt.Errorf("check file artifact %s: %w", artifact.Reference, err)
			}
			location := filepath.Join(e.workspace.ArtifactsPath(run.Id), artifact.Reference)
			if !exists {
				e.logger.Warn("artifact file not found, skipping", "run", run.Id, "stage", stage.Name, "location", location)
				continue
			}
			created := artifactForRun(run, stage, artifact)
			created.Location = &location
			if err := e.createArtifact(ctx, created); err != nil {
				return err
			}
		case "command":
			if err := e.archiveCommandArtifact(ctx, run, stage, artifact, runOptions); err != nil {
				return err
			}
		case "docker_image":
			if err := e.archiveContainerImageArtifact(ctx, run, stage, stages, artifact, runtimeDatetime); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported artifact collector %s", artifact.Collector)
		}
	}
	return nil
}

func (e Executor) archiveCommandArtifact(ctx context.Context, run model.PipelineRun, stage model.StageDefinition, artifact model.ArtifactConfig, runOptions pipelinerunport.RunOptions) error {
	runner, ok := e.runner.(pipelinerunport.CommandOutputRunner)
	if !ok {
		return fmt.Errorf("command artifact execution is unavailable")
	}
	output, err := runner.RunCommand(ctx, runOptions, artifact.Command)
	if err != nil {
		return fmt.Errorf("collect command artifact %s: %w", artifact.Name, err)
	}
	value := strings.TrimSpace(output)
	if artifact.Format == "git_object_id" && !gitObjectIdPattern.MatchString(value) {
		return fmt.Errorf("command artifact %s must be a 40 or 64 character hexadecimal Git object Id", artifact.Name)
	}
	created := artifactForRun(run, stage, artifact)
	created.Value = &value
	created.ValueFormat = &artifact.Format
	if err := e.createArtifact(ctx, created); err != nil {
		return err
	}
	return nil
}

func (e Executor) archiveContainerImageArtifact(ctx context.Context, run model.PipelineRun, stage model.StageDefinition, stages map[string]model.StageDefinition, artifact model.ArtifactConfig, _ string) error {
	inspector, ok := e.runner.(pipelinerunport.ImageInspector)
	if !ok {
		return fmt.Errorf("local image inspection is unavailable")
	}
	imageId, err := inspector.ImageId(ctx, artifact.Reference)
	if err != nil {
		return fmt.Errorf("inspect local image %s: %w", artifact.Reference, err)
	}
	created := artifactForRun(run, stage, artifact)
	created.ImageRef = &artifact.Reference
	created.LocalImageSha256 = &imageId
	if artifact.ComponentName != nil {
		source, err := sourceCommitArtifactForStage(stage, stages)
		if err != nil {
			return err
		}
		sourceArtifact, err := e.store.CommandArtifactByRunStageAndName(ctx, run.Id, source.ProducerStageId, source.Artifact.Name)
		if err != nil {
			return fmt.Errorf("load source command artifact: %w", err)
		}
		if sourceArtifact.Value == nil || sourceArtifact.ValueFormat == nil || *sourceArtifact.ValueFormat != "git_object_id" || !gitObjectIdPattern.MatchString(*sourceArtifact.Value) {
			return fmt.Errorf("source command artifact %s is not a valid Git object Id", source.Artifact.Name)
		}
		created.SourceArtifactId = &sourceArtifact.Id
		created.SourceCommitSha = sourceArtifact.Value
	}
	return e.createArtifact(ctx, created)
}

type forkBuildVersionStore interface {
	PipelineRunVersionBinding(ctx context.Context, pipelineRunId string) (model.PipelineRunVersionBinding, error)
	ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error)
	CompletePipelineRunVersionBinding(ctx context.Context, pipelineRunId string, generatedVersionId string, generatedVersionLabel string) error
}

func (e Executor) forkBuildVersion(ctx context.Context, run model.PipelineRun, stages []model.StageDefinition, runtimeDatetime string) error {
	return forkBuildVersion(ctx, e.store, e.transactionRunner, e.versionForker, run, stages, runtimeDatetime)
}

func forkBuildVersion(ctx context.Context, store forkBuildVersionStore, transactionRunner pipelinerunport.TransactionRunner, versionForker applicationport.BuildVersionForker, run model.PipelineRun, stages []model.StageDefinition, runtimeDatetime string) error {
	binding, err := store.PipelineRunVersionBinding(ctx, run.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("load run version binding: %w", err)
	}
	if binding.GeneratedVersionId != nil {
		return nil
	}
	if versionForker == nil {
		return fmt.Errorf("application version forker is unavailable")
	}
	artifacts, err := store.ListArtifactsByRun(ctx, run.ProjectId, run.Id)
	if err != nil {
		return fmt.Errorf("load pipeline artifacts: %w", err)
	}
	byStageArtifact := map[string]model.Artifact{}
	for _, artifact := range artifacts {
		byStageArtifact[artifact.PipelineStageId+"\x00"+artifact.Name] = artifact
	}
	updates := make([]applicationport.BuildVersionComponentUpdate, 0)
	for _, mapping := range componentMappings(stages) {
		artifact, exists := byStageArtifact[mapping.Stage.Id+"\x00"+mapping.Artifact.Name]
		if !exists || artifact.ImageRef == nil || artifact.LocalImageSha256 == nil || artifact.SourceCommitSha == nil {
			return fmt.Errorf("component-bound artifact %s from stage %s was not collected", mapping.Artifact.Name, mapping.Stage.Name)
		}
		updates = append(updates, applicationport.BuildVersionComponentUpdate{ComponentName: mapping.ComponentName, Image: *artifact.ImageRef, ArtifactId: artifact.Id, ArtifactName: artifact.Name, LocalImageSha256: *artifact.LocalImageSha256, SourceCommitSha: *artifact.SourceCommitSha})
	}
	if len(updates) == 0 {
		return nil
	}
	if transactionRunner == nil {
		return fmt.Errorf("transaction runner is unavailable")
	}
	return transactionRunner.RunInTransaction(ctx, func(txCtx context.Context) error {
		version, err := versionForker.ForkVersionForBuild(txCtx, applicationport.BuildVersionForkInput{
			SourceVersionId: binding.SourceVersionId,
			Label:           buildVersionLabel(runtimeDatetime),
			Components:      updates,
		})
		if err != nil {
			return fmt.Errorf("fork source version: %w", err)
		}
		return store.CompletePipelineRunVersionBinding(txCtx, run.Id, version.Id, version.Label)
	})
}

func buildVersionLabel(runtimeDatetime string) string {
	return "build-" + runtimeDatetime
}

type sourceCommitArtifact struct {
	ProducerStageId string
	Artifact        model.ArtifactConfig
}

func sourceCommitArtifactForStage(stage model.StageDefinition, stages map[string]model.StageDefinition) (sourceCommitArtifact, error) {
	seen := map[string]bool{}
	candidates := make([]sourceCommitArtifact, 0, 1)
	var visit func(string) error
	visit = func(stageId string) error {
		if seen[stageId] {
			return nil
		}
		seen[stageId] = true
		ancestor, exists := stages[stageId]
		if !exists {
			return fmt.Errorf("dependency %s does not exist", stageId)
		}
		for _, artifact := range ancestor.Artifacts {
			if artifact.Collector == "command" && artifact.Format == "git_object_id" {
				candidates = append(candidates, sourceCommitArtifact{ProducerStageId: ancestor.Id, Artifact: artifact})
			}
		}
		for _, dependency := range ancestor.DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		return nil
	}
	for _, dependency := range stage.DependsOn {
		if err := visit(dependency); err != nil {
			return sourceCommitArtifact{}, err
		}
	}
	if len(candidates) != 1 {
		return sourceCommitArtifact{}, fmt.Errorf("requires exactly one transitive git_object_id artifact, found %d", len(candidates))
	}
	return candidates[0], nil
}

func artifactForRun(run model.PipelineRun, stage model.StageDefinition, config model.ArtifactConfig) model.Artifact {
	return model.Artifact{
		Id: idutil.NewId(), ProjectId: run.ProjectId, PipelineRunId: run.Id, RepositoryId: run.RepositoryId, RepositoryName: run.RepositoryName,
		PipelineId: run.PipelineId, PipelineName: run.PipelineName, PipelineStageId: stage.Id, StageName: stage.Name,
		Collector: config.Collector, Name: config.Name, CreatedAt: now(),
	}
}

func (e Executor) createArtifact(ctx context.Context, artifact model.Artifact) error {
	if err := validateArtifactPayload(artifact); err != nil {
		return err
	}
	return e.store.CreateArtifact(ctx, artifact)
}

func validateArtifactPayload(artifact model.Artifact) error {
	switch artifact.Collector {
	case "file":
		if artifact.Location == nil || artifact.Value != nil || artifact.ValueFormat != nil || artifact.ImageRef != nil || artifact.LocalImageSha256 != nil {
			return fmt.Errorf("invalid file artifact payload")
		}
	case "command":
		if artifact.Location != nil || artifact.Value == nil || artifact.ImageRef != nil || artifact.LocalImageSha256 != nil || artifact.ValueFormat == nil || (*artifact.ValueFormat != "text" && *artifact.ValueFormat != "git_object_id") {
			return fmt.Errorf("invalid command artifact payload")
		}
	case "docker_image":
		if artifact.Location != nil || artifact.Value != nil || artifact.ValueFormat != nil || artifact.ImageRef == nil || artifact.LocalImageSha256 == nil {
			return fmt.Errorf("invalid docker image artifact payload")
		}
	default:
		return fmt.Errorf("unsupported artifact collector %s", artifact.Collector)
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
