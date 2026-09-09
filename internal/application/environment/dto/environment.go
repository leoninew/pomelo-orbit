package dto

type UpdateInput struct {
	State      *string
	TargetType *string
	SSH        *SSHTargetInput
}

type SSHTargetInput struct {
	Platform      string
	Host          string
	Port          int
	Username      string
	WorkspaceRoot string
}
