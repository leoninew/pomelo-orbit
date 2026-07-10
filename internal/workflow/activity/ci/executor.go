package cisvc

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/ciworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Executor struct {
	store     repository.PipelineExecutionStore
	workspace *ciworkspace.Workspace
	logStore  ExecutionLogStore
	secretKey string
	logger    *slog.Logger
	runner    ContainerRunner
}

func (e Executor) Execute(ctx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, stages []model.StageDefinition) (bool, string) {
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
		results := e.executeLayer(ctx, run, repo, variables, layer, stageById)
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

func (e Executor) executeLayer(ctx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, layer []string, stages map[string]model.StageDefinition) map[string]bool {
	results := make(map[string]bool, len(layer))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, stageId := range layer {
		stage := stages[stageId]
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok := e.executeStage(ctx, run, repo, variables, stage)
			mu.Lock()
			results[stage.Name] = ok
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results
}

func (e Executor) executeStage(ctx context.Context, run model.PipelineRun, repo model.Repository, variables map[string]any, stage model.StageDefinition) bool {
	stageRun := model.StageRun{Id: idutil.NewId(), PipelineRunId: run.Id, StageId: stage.Id, StageName: stage.Name, Status: status.WorkStatusWaitingToRun}
	if err := e.store.InsertStageRun(ctx, stageRun); err != nil {
		e.logger.Error("stage run insert failed", "run", run.Id, "stage", stage.Name, "error", err)
		return false
	}

	started := now()
	stageRun.StartedAt = &started
	stageRun.Status = status.WorkStatusRunning
	_ = e.store.UpdateStageRun(ctx, stageRun)

	logPath := e.workspace.StageLogPath(run.Id, stageRun.Id)
	logWriter, err := e.logStore.Writer(logPath)
	if err != nil {
		return e.failStage(ctx, stageRun, fmt.Sprintf("create log writer: %v", err))
	}
	defer func() { _ = logWriter.Close() }()

	volumes, err := e.workspace.DockerStageMounts(ctx, repo.Code, run.Id)
	if err != nil {
		return e.failStage(ctx, stageRun, err.Error())
	}

	script, environment, err := e.stageRunConfig(ctx, repo, variables, stage)
	if err != nil {
		return e.failStage(ctx, stageRun, err.Error())
	}

	exitCode, _, err := e.runner.Run(ctx, RunOptions{
		Image:       stage.Image,
		Script:      script,
		Environment: environment,
		Volumes:     volumes,
		LogWriter:   logWriter,
	})
	if err != nil {
		return e.failStage(ctx, stageRun, err.Error())
	}
	if exitCode != 0 {
		if err := logWriter.Close(); err != nil {
			return e.failStage(ctx, stageRun, fmt.Sprintf("close stage log writer: %v", err))
		}
		content, _, _ := e.logStore.Read(logPath, 0)
		errMsg := lastLines(string(content), 3)
		return e.completeStageFailed(ctx, stageRun, exitCode, errMsg)
	}

	finished := now()
	stageRun.FinishedAt = &finished
	stageRun.Status = status.WorkStatusRanToCompletion
	stageRun.ExitCode = &exitCode
	if err := e.saveArtifacts(ctx, run, stage); err != nil {
		return e.failStage(ctx, stageRun, err.Error())
	}
	if err := e.store.UpdateStageRun(ctx, stageRun); err != nil {
		e.logger.Error("stage run update failed", "run", run.Id, "stage", stage.Name, "error", err)
		return false
	}
	e.logger.Info("stage succeeded", "run", run.Id, "stage", stage.Name)
	return true
}

func (e Executor) stageRunConfig(ctx context.Context, repo model.Repository, variables map[string]any, stage model.StageDefinition) (string, []string, error) {
	script := safeCommand(commandLines(stage.Script))
	environment := envMap(variables)
	if repo.GitCredentialId == nil || !stageUsesRepositoryURL(stage.Script, repo.RepositoryURL) {
		return script, environment, nil
	}
	credential, err := e.store.Credential(ctx, *repo.GitCredentialId)
	if err != nil {
		return "", nil, fmt.Errorf("load git credential: %w", err)
	}
	authenticatedURL, err := e.authenticatedRepositoryURL(repo.RepositoryURL, credential)
	if err != nil {
		return "", nil, err
	}
	environment = append(environment, gitCredentialEnvironment(repo.RepositoryURL, authenticatedURL)...)
	return script, environment, nil
}

func stageUsesRepositoryURL(script string, repositoryURL string) bool {
	return strings.TrimSpace(repositoryURL) != "" && strings.Contains(script, repositoryURL)
}

func (e Executor) authenticatedRepositoryURL(repositoryURL string, credential model.Credential) (string, error) {
	credentialData, err := security.DecryptString(e.secretKey, credential.EncryptedData)
	if err != nil {
		return "", fmt.Errorf("decrypt git credential: %w", err)
	}
	switch credential.Type {
	case "github_token":
		return buildAuthenticatedRepositoryURL(repositoryURL, credentialData)
	case "gitee_token":
		username, token, err := splitGiteeCredential(credentialData)
		if err != nil {
			return "", err
		}
		return buildAuthenticatedRepositoryURL(repositoryURL, username+":"+token)
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

func buildAuthenticatedRepositoryURL(repositoryURL string, authPart string) (string, error) {
	if strings.TrimSpace(authPart) == "" {
		return "", fmt.Errorf("git credential data is empty")
	}
	parsed, err := url.Parse(repositoryURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("unsupported repository URL format: %s", repositoryURL)
	}
	if strings.Contains(authPart, ":") {
		username, password, _ := strings.Cut(authPart, ":")
		parsed.User = url.UserPassword(username, password)
	} else {
		parsed.User = url.User(authPart)
	}
	return parsed.String(), nil
}

func gitCredentialEnvironment(repositoryURL string, authenticatedURL string) []string {
	return []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=url." + authenticatedURL + ".insteadOf",
		"GIT_CONFIG_VALUE_0=" + repositoryURL,
	}
}

func (e Executor) completeStageFailed(ctx context.Context, stageRun model.StageRun, exitCode int, message string) bool {
	finished := now()
	stageRun.FinishedAt = &finished
	stageRun.Status = status.WorkStatusFaulted
	stageRun.ExitCode = &exitCode
	stageRun.ErrorMessage = &message
	_ = e.store.UpdateStageRun(ctx, stageRun)
	e.logger.Warn("stage failed", "run", stageRun.PipelineRunId, "stage", stageRun.StageName, "exit_code", exitCode)
	return false
}

func (e Executor) failStage(ctx context.Context, stageRun model.StageRun, message string) bool {
	finished := now()
	stageRun.FinishedAt = &finished
	stageRun.Status = status.WorkStatusFaulted
	stageRun.ErrorMessage = &message
	_ = e.store.UpdateStageRun(ctx, stageRun)
	e.logger.Error("stage faulted", "run", stageRun.PipelineRunId, "stage", stageRun.StageName, "error", message)
	return false
}

func (e Executor) cancelRemaining(ctx context.Context, runId string, layers [][]string, start int, stages map[string]model.StageDefinition) {
	for i := start; i < len(layers); i++ {
		for _, stageId := range layers[i] {
			stage := stages[stageId]
			_ = e.store.InsertStageRun(ctx, model.StageRun{Id: idutil.NewId(), PipelineRunId: runId, StageId: stage.Id, StageName: stage.Name, Status: status.WorkStatusCanceled})
		}
	}
}

func (e Executor) saveArtifacts(ctx context.Context, run model.PipelineRun, stage model.StageDefinition) error {
	artifactRoot := e.workspace.ArtifactsPath(run.Id)
	for _, artifact := range stage.Artifacts {
		artifactPath := artifact.Path
		if artifact.Type == "binary" {
			artifactPath = filepath.Join(artifactRoot, artifact.Path)
			if _, err := os.Stat(artifactPath); err != nil {
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
