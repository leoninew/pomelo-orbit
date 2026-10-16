package dto

type ProjectSummary struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	IsActive      bool   `json:"is_active"`
	EnvironmentId string `json:"environment_id"`
	TargetType    string `json:"target_type"`
	Platform      string `json:"platform"`
	Host          string `json:"host"`
	Username      string `json:"username"`
	Port          int    `json:"port"`
}

type CommandInput struct {
	ProjectCode    string
	Argv           []string
	Workdir        string
	Stdin          []byte
	TimeoutSeconds *uint32
}

type CommandResult struct {
	ProjectId       string `json:"project_id"`
	ProjectCode     string `json:"project_code"`
	EnvironmentId   string `json:"environment_id"`
	TargetRevision  int64  `json:"target_revision"`
	State           string `json:"state"`
	ExitCode        *int32 `json:"exit_code"`
	Stdout          []byte `json:"stdout"`
	Stderr          []byte `json:"stderr"`
	DurationMs      int64  `json:"duration_ms"`
	OutputTruncated bool   `json:"output_truncated"`
}
