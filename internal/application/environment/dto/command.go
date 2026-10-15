package dto

type ProjectSummary struct {
	Id, Name, Code                                      string
	IsActive                                            bool
	EnvironmentId, TargetType, Platform, Host, Username string
	Port                                                int
}

type CommandInput struct {
	ProjectCode    string
	Argv           []string
	Workdir        string
	Stdin          []byte
	TimeoutSeconds *uint32
}

type CommandResult struct {
	ProjectId, ProjectCode, EnvironmentId string
	TargetRevision                        int64
	State                                 string
	ExitCode                              *int32
	Stdout, Stderr                        []byte
	DurationMs                            int64
	OutputTruncated                       bool
}
