package port

import (
	"context"
	"io"

	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
)

type PipelineRunDispatcher interface {
	DispatchPipelineRun(ctx context.Context, input pipelinerundto.PipelineRunDispatchInput) error
}

// TransactionRunner groups a short application-level database write segment.
type TransactionRunner interface {
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
}

type LogReader interface {
	Read(logPath string, offset int) ([]byte, int, error)
}

type ExecutionLogStore interface {
	LogReader
	Writer(logPath string) (io.WriteCloser, error)
}

type VolumeMount struct {
	HostPath      string
	ContainerPath string
	Mode          string
}

type Workspace interface {
	CreateRunDirectories(projectCode string, runId string) error
	ArtifactsPath(runId string) string
	ArtifactExists(runId string, artifactPath string) (bool, error)
	StageLogPath(runId string, pipelineStageRunId string) string
	DockerStageMounts(ctx context.Context, projectCode string, runId string) ([]VolumeMount, error)
}

type RunOptions struct {
	ContainerName string
	Image         string
	Script        string
	Environment   []string
	Volumes       []VolumeMount
	LogWriter     io.Writer
}

type ContainerRunner interface {
	Run(ctx context.Context, opts RunOptions) (int, string, error)
}

// CommandOutputRunner runs a collector command in the same image, environment, and mounts as its stage.
type CommandOutputRunner interface {
	RunCommand(ctx context.Context, opts RunOptions, command string) (string, error)
}

type ImageInspector interface {
	ImageId(ctx context.Context, imageRef string) (string, error)
}
