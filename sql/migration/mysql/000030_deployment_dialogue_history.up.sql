CREATE TABLE deployment_dialogue_conversation (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26) NOT NULL,
    created_by_user_id VARCHAR(26) NOT NULL,
    title VARCHAR(120) NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    INDEX idx_deployment_dialogue_conversation_project_updated (project_id, updated_at DESC, id DESC)
);

CREATE TABLE deployment_dialogue_message (
    id VARCHAR(26) PRIMARY KEY,
    conversation_id VARCHAR(26) NOT NULL,
    role VARCHAR(16) NOT NULL,
    content LONGTEXT NOT NULL,
    created_at DATETIME NOT NULL,
    CONSTRAINT chk_deployment_dialogue_message_role CHECK (role IN ('user', 'assistant')),
    INDEX idx_deployment_dialogue_message_conversation_created (conversation_id, created_at, id)
);
