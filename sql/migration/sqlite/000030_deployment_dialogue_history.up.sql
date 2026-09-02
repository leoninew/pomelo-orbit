CREATE TABLE deployment_dialogue_conversation (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE TABLE deployment_dialogue_message (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (conversation_id) REFERENCES deployment_dialogue_conversation(id) ON DELETE CASCADE
);

CREATE INDEX idx_deployment_dialogue_conversation_project_updated
    ON deployment_dialogue_conversation(project_id, updated_at DESC, id DESC);
CREATE INDEX idx_deployment_dialogue_message_conversation_created
    ON deployment_dialogue_message(conversation_id, created_at ASC, id ASC);
