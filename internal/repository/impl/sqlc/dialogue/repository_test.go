package dialoguerepo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
	_ "modernc.org/sqlite"
)

func TestRepositoryListsMessagesInOrderAndDeletesConversationCascade(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE user (id TEXT PRIMARY KEY);
		CREATE TABLE project (id TEXT PRIMARY KEY);
		CREATE TABLE deployment_dialogue_conversation (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			created_by_user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
			FOREIGN KEY (created_by_user_id) REFERENCES user(id) ON DELETE RESTRICT
		);
		CREATE TABLE deployment_dialogue_message (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (conversation_id) REFERENCES deployment_dialogue_conversation(id) ON DELETE CASCADE
		);
		INSERT INTO user (id) VALUES ('user-1');
		INSERT INTO project (id) VALUES ('project-1'), ('project-2');
	`); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	repository := NewRepository(database)
	createdAt := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	conversations := []model.DeploymentDialogueConversation{
		{Id: "conversation-1", ProjectId: "project-1", CreatedByUserId: "user-1", Title: "早些的对话", CreatedAt: createdAt, UpdatedAt: createdAt},
		{Id: "conversation-2", ProjectId: "project-1", CreatedByUserId: "user-1", Title: "较新的对话", CreatedAt: createdAt, UpdatedAt: createdAt.Add(time.Minute)},
		{Id: "conversation-3", ProjectId: "project-2", CreatedByUserId: "user-1", Title: "另一项目", CreatedAt: createdAt, UpdatedAt: createdAt.Add(2 * time.Minute)},
	}
	for _, conversation := range conversations {
		if err := repository.CreateDeploymentDialogueConversation(ctx, conversation); err != nil {
			t.Fatalf("create conversation %s: %v", conversation.Id, err)
		}
	}
	for _, message := range []model.DeploymentDialogueMessage{
		{Id: "message-2", ConversationId: "conversation-2", Role: "assistant", Content: "回答", CreatedAt: createdAt},
		{Id: "message-1", ConversationId: "conversation-2", Role: "user", Content: "问题", CreatedAt: createdAt},
	} {
		if err := repository.CreateDeploymentDialogueMessage(ctx, message); err != nil {
			t.Fatalf("create message %s: %v", message.Id, err)
		}
	}

	items, err := repository.ListDeploymentDialogueConversations(ctx, "project-1")
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(items) != 2 || items[0].Id != "conversation-2" || items[1].Id != "conversation-1" {
		t.Fatalf("conversations = %#v", items)
	}
	messages, err := repository.ListDeploymentDialogueMessages(ctx, "conversation-2")
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 || messages[0].Id != "message-1" || messages[1].Id != "message-2" {
		t.Fatalf("messages = %#v", messages)
	}

	if err := repository.DeleteDeploymentDialogueConversation(ctx, "conversation-2"); err != nil {
		t.Fatalf("delete conversation: %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM deployment_dialogue_message WHERE conversation_id = 'conversation-2'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("conversation messages = %d, want 0", count)
	}
}
