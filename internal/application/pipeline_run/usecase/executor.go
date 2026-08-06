package pipelinerunsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	applicationport "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/port"
	pipelinerunport "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/port"
	repositoryport "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

var gitObjectIdPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

type buildArtifactStore interface {
	pipelineExecutionStore
	PipelineRunBuildVersionBinding(ctx context.Context, pipelineRunId string, pipelineStageId string) (model.PipelineRunBuildVersionBinding, error)
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
	CompletePipelineRunBuildVersionBinding(ctx context.Context, pipelineRunId string, pipelineStageId string, generatedVersionId string, generatedVersionLabel string, artifactId string) error
}

type Executor struct {
	store            pipelineExecutionStore
	versionForker    applicationport.BuildVersionForker
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
			ok := e.executeStage(ctx, executionCtx, run, repo, variables, stage, stages)
			mu.Lock()
			results[stage.Name] = ok
			mu.Unlock()
		})
	}
	wg.Wait()
	return results
}

func (e Executor) executeStage(ctx context.Context, executionCtx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, stage model.StageDefinition, stages map[string]model.StageDefinition) bool {
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
			if err := e.store.CreateArtifact(ctx, created); err != nil {
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
	if err := e.store.CreateArtifact(ctx, created); err != nil {
		return err
	}
	return nil
}

func (e Executor) archiveContainerImageArtifact(ctx context.Context, run model.PipelineRun, stage model.StageDefinition, stages map[string]model.StageDefinition, artifact model.ArtifactConfig, runtimeDatetime string) error {
	store, ok := e.store.(buildArtifactStore)
	if !ok {
		return fmt.Errorf("container image artifact storage is unavailable")
	}
	inspector, ok := e.runner.(pipelinerunport.ImageInspector)
	if !ok {
		return fmt.Errorf("local image inspection is unavailable")
	}
	sourceCommitArtifact, err := sourceCommitArtifactForStage(stage, stages)
	if err != nil {
		return err
	}
	sourceArtifact, err := store.CommandArtifactByRunStageAndName(ctx, run.Id, sourceCommitArtifact.ProducerStageId, sourceCommitArtifact.Artifact.Name)
	if err != nil {
		return fmt.Errorf("load source command artifact: %w", err)
	}
	if sourceArtifact.Value == nil || sourceArtifact.ValueFormat == nil || *sourceArtifact.ValueFormat != "git_object_id" || !gitObjectIdPattern.MatchString(*sourceArtifact.Value) {
		return fmt.Errorf("source command artifact %s is not a valid Git object Id", sourceCommitArtifact.Artifact.Name)
	}
	imageId, err := inspector.ImageId(ctx, artifact.Reference)
	if err != nil {
		return fmt.Errorf("inspect local image %s: %w", artifact.Reference, err)
	}
	if stage.BuildVersionBinding != nil {
		binding, err := store.PipelineRunBuildVersionBinding(ctx, run.Id, stage.Id)
		if err != nil {
			return fmt.Errorf("load build version binding: %w", err)
		}
		if binding.GeneratedVersionId != nil {
			return nil
		}
	}
	return store.RunInTransaction(ctx, func(txCtx context.Context) error {
		created := artifactForRun(run, stage, artifact)
		created.ImageRef = &artifact.Reference
		created.LocalImageSha256 = &imageId
		created.SourceArtifactId = &sourceArtifact.Id
		created.SourceCommitSha = sourceArtifact.Value
		if err := store.CreateArtifact(txCtx, created); err != nil {
			return err
		}
		if stage.BuildVersionBinding == nil {
			return nil
		}
		return e.forkBuildVersion(txCtx, store, run, stage, created, runtimeDatetime)
	})
}

func (e Executor) forkBuildVersion(ctx context.Context, store buildArtifactStore, run model.PipelineRun, stage model.StageDefinition, artifact model.Artifact, runtimeDatetime string) error {
	binding, err := store.PipelineRunBuildVersionBinding(ctx, run.Id, stage.Id)
	if err != nil {
		return fmt.Errorf("load build version binding: %w", err)
	}
	if binding.GeneratedVersionId != nil {
		return nil
	}
	if e.versionForker == nil {
		return fmt.Errorf("application version forker is unavailable")
	}
	version, err := e.versionForker.ForkVersionForBuild(ctx, applicationport.BuildVersionForkInput{
		SourceVersionId: binding.SourceVersionId,
		Label:           buildVersionLabel(runtimeDatetime),
		ComponentName:   binding.ComponentName,
		Image:           *artifact.ImageRef,
		ArtifactId:      artifact.Id,
	})
	if err != nil {
		return fmt.Errorf("fork source version: %w", err)
	}
	if err := store.CompletePipelineRunBuildVersionBinding(ctx, run.Id, stage.Id, version.Id, version.Label, artifact.Id); err != nil {
		return err
	}
	return nil
}

func buildVersionLabel(runtimeDatetime string) string {
	return "build-" + runtimeDatetime
}

func artifactForRun(run model.PipelineRun, stage model.StageDefinition, config model.ArtifactConfig) model.Artifact {
	return model.Artifact{
		Id: idutil.NewId(), ProjectId: run.ProjectId, PipelineRunId: run.Id, RepositoryId: run.RepositoryId, RepositoryName: run.RepositoryName,
		TemplateId: run.TemplateId, TemplateName: run.TemplateName, PipelineStageId: stage.Id, StageName: stage.Name,
		Collector: config.Collector, Name: config.Name, CreatedAt: now(),
	}
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
