package ci

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/repository/model"
	"backend/internal/status"
)

type Executor struct {
	store  Store
	cfg    config.Config
	logger *slog.Logger
	runner ContainerRunner
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
	stageRun := model.StageRun{Id: repository.NewId(), PipelineRunId: run.Id, StageId: stage.Id, StageName: stage.Name, Status: status.WorkStatusWaitingToRun}
	if err := e.store.InsertStageRun(ctx, stageRun); err != nil {
		e.logger.Error("stage run insert failed", "run", run.Id, "stage", stage.Name, "error", err)
		return false
	}

	started := now()
	stageRun.StartedAt = &started
	stageRun.Status = status.WorkStatusRunning
	_ = e.store.UpdateStageRun(ctx, stageRun)

	logPath := filepath.Join(e.cfg.DataRoot(), "ci", "runs", run.Id, "stages", stageRun.Id+".log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return e.failStage(ctx, stageRun, fmt.Sprintf("create stage log dir: %v", err))
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return e.failStage(ctx, stageRun, fmt.Sprintf("create stage log: %v", err))
	}
	defer func() { _ = logFile.Close() }()

	exitCode, output, err := e.runner.Run(ctx, RunOptions{
		Image:       stage.Image,
		Script:      safeCommand(commandLines(stage.Script)),
		Environment: envMap(variables),
		Volumes: []VolumeMount{
			{HostPath: filepath.Join(e.cfg.DataRoot(), "ci", repo.Code, "workspace"), ContainerPath: "/workspace", Mode: "rw"},
			{HostPath: filepath.Join(e.cfg.DataRoot(), "ci", "runs", run.Id, "artifacts"), ContainerPath: "/artifacts", Mode: "rw"},
		},
		LogFile: logFile,
	})
	if err != nil {
		return e.failStage(ctx, stageRun, err.Error())
	}
	if exitCode != 0 {
		return e.completeStageFailed(ctx, stageRun, exitCode, lastLines(output, 3))
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
			_ = e.store.InsertStageRun(ctx, model.StageRun{Id: repository.NewId(), PipelineRunId: runId, StageId: stage.Id, StageName: stage.Name, Status: status.WorkStatusCanceled})
		}
	}
}

func (e Executor) saveArtifacts(ctx context.Context, run model.PipelineRun, stage model.StageDefinition) error {
	artifactRoot := filepath.Join(e.cfg.DataRoot(), "ci", "runs", run.Id, "artifacts")
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
