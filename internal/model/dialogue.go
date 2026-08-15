package model

import "time"

type DeploymentDialogueConversation struct {
	Id              string
	ProjectId       string
	CreatedByUserId string
	Title           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type DeploymentDialogueMessage struct {
	Id             string
	ConversationId string
	Role           string
	Content        string
	CreatedAt      time.Time
}
