package taskhandler

import "encoding/json"

type CreateTaskReq struct {
	Id          string          `json:"id"`
	TaskType    string          `json:"task_type"`
	Payload     json.RawMessage `json:"payload"`
	PayloadJSON string          `json:"payload_json"`
	MaxAttempts int             `json:"max_attempts"`
}

type TaskResp struct {
	Id           string  `json:"id"`
	TaskType     string  `json:"task_type"`
	PayloadJSON  string  `json:"payload_json"`
	Status       string  `json:"status"`
	Attempts     int     `json:"attempts"`
	MaxAttempts  int     `json:"max_attempts"`
	LockedBy     *string `json:"locked_by"`
	LockedAt     *string `json:"locked_at"`
	StartedAt    *string `json:"started_at"`
	FinishedAt   *string `json:"finished_at"`
	ErrorMessage *string `json:"error_message"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}
