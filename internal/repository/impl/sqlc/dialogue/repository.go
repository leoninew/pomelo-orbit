package dialoguerepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dialoguesqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/dialogue"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.DeploymentDialogueStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *dialoguesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *dialoguesqlc.Queries {
		return dialoguesqlc.New(dbtx)
	})
}

func (r Repository) ListDeploymentDialogueConversations(ctx context.Context, projectId string) ([]model.DeploymentDialogueConversation, error) {
	rows, err := r.q(ctx).ListDeploymentDialogueConversations(ctx, projectId)
	if err != nil {
		return nil, fmt.Errorf("list deployment dialogue conversations: %w", err)
	}
	items := make([]model.DeploymentDialogueConversation, 0, len(rows))
	for _, row := range rows {
		items = append(items, conversationFrom(row))
	}
	return items, nil
}

func (r Repository) DeploymentDialogueConversation(ctx context.Context, id string) (model.DeploymentDialogueConversation, error) {
	row, err := r.q(ctx).DeploymentDialogueConversationByID(ctx, id)
	if err != nil {
		return model.DeploymentDialogueConversation{}, fmt.Errorf("load deployment dialogue conversation %s: %w", id, sqlcommon.TranslateError(err))
	}
	return conversationFrom(row), nil
}

func (r Repository) ListDeploymentDialogueMessages(ctx context.Context, conversationId string) ([]model.DeploymentDialogueMessage, error) {
	rows, err := r.q(ctx).ListDeploymentDialogueMessages(ctx, conversationId)
	if err != nil {
		return nil, fmt.Errorf("list deployment dialogue messages: %w", err)
	}
	items := make([]model.DeploymentDialogueMessage, 0, len(rows))
	for _, row := range rows {
		items = append(items, messageFrom(row))
	}
	return items, nil
}

func (r Repository) CreateDeploymentDialogueConversation(ctx context.Context, conversation model.DeploymentDialogueConversation) error {
	if err := r.q(ctx).CreateDeploymentDialogueConversation(ctx, dialoguesqlc.CreateDeploymentDialogueConversationParams{
		ID:              conversation.Id,
		ProjectID:       conversation.ProjectId,
		CreatedByUserID: conversation.CreatedByUserId,
		Title:           conversation.Title,
		CreatedAt:       conversation.CreatedAt,
		UpdatedAt:       conversation.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("create deployment dialogue conversation: %w", err)
	}
	return nil
}

func (r Repository) CreateDeploymentDialogueMessage(ctx context.Context, message model.DeploymentDialogueMessage) error {
	if err := r.q(ctx).CreateDeploymentDialogueMessage(ctx, dialoguesqlc.CreateDeploymentDialogueMessageParams{
		ID:             message.Id,
		ConversationID: message.ConversationId,
		Role:           message.Role,
		Content:        message.Content,
		CreatedAt:      message.CreatedAt,
	}); err != nil {
		return fmt.Errorf("create deployment dialogue message: %w", err)
	}
	return nil
}

func (r Repository) TouchDeploymentDialogueConversation(ctx context.Context, id string, updatedAt time.Time) error {
	if err := r.q(ctx).TouchDeploymentDialogueConversation(ctx, dialoguesqlc.TouchDeploymentDialogueConversationParams{ID: id, UpdatedAt: updatedAt}); err != nil {
		return fmt.Errorf("touch deployment dialogue conversation %s: %w", id, err)
	}
	return nil
}

func (r Repository) DeleteDeploymentDialogueConversation(ctx context.Context, id string) error {
	if err := r.q(ctx).DeleteDeploymentDialogueConversation(ctx, id); err != nil {
		return fmt.Errorf("delete deployment dialogue conversation %s: %w", id, err)
	}
	return nil
}

func conversationFrom(row dialoguesqlc.DeploymentDialogueConversation) model.DeploymentDialogueConversation {
	return model.DeploymentDialogueConversation{
		Id:              row.ID,
		ProjectId:       row.ProjectID,
		CreatedByUserId: row.CreatedByUserID,
		Title:           row.Title,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func messageFrom(row dialoguesqlc.DeploymentDialogueMessage) model.DeploymentDialogueMessage {
	return model.DeploymentDialogueMessage{
		Id:             row.ID,
		ConversationId: row.ConversationID,
		Role:           row.Role,
		Content:        row.Content,
		CreatedAt:      row.CreatedAt,
	}
}
