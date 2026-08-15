package repository

import (
	"context"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

type DeploymentDialogueStore interface {
	ListDeploymentDialogueConversations(ctx context.Context, projectId string) ([]model.DeploymentDialogueConversation, error)
	DeploymentDialogueConversation(ctx context.Context, id string) (model.DeploymentDialogueConversation, error)
	ListDeploymentDialogueMessages(ctx context.Context, conversationId string) ([]model.DeploymentDialogueMessage, error)
	CreateDeploymentDialogueConversation(ctx context.Context, conversation model.DeploymentDialogueConversation) error
	CreateDeploymentDialogueMessage(ctx context.Context, message model.DeploymentDialogueMessage) error
	TouchDeploymentDialogueConversation(ctx context.Context, id string, updatedAt time.Time) error
	DeleteDeploymentDialogueConversation(ctx context.Context, id string) error
}
