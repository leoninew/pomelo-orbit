-- v0.7.3: 项目表

CREATE TABLE IF NOT EXISTS project (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (owner_user_id) REFERENCES user(id) ON DELETE CASCADE,
    UNIQUE(owner_user_id, code)
);

CREATE INDEX IF NOT EXISTS idx_project_owner_user_id ON project(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_project_code ON project(code);

INSERT INTO project (id, name, code, owner_user_id, created_at, updated_at)
SELECT
    '01PROJECTDEFAULT0000000000',
    'Default Project',
    'default',
    id,
    datetime('now'),
    datetime('now')
FROM user
WHERE username = 'admin'
  AND NOT EXISTS (SELECT 1 FROM project WHERE id = '01PROJECTDEFAULT0000000000')
LIMIT 1;

ALTER TABLE credential ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE pipeline_template ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE build_stage ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE pipeline_snapshot ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE repository ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE pipeline_run ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE artifact ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE application ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE deployment ADD COLUMN project_id TEXT REFERENCES project(id);
ALTER TABLE route ADD COLUMN project_id TEXT REFERENCES project(id);

UPDATE credential SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE pipeline_template SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE build_stage SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE pipeline_snapshot SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE repository SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE pipeline_run SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE artifact SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE application SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE deployment SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;
UPDATE route SET project_id = '01PROJECTDEFAULT0000000000' WHERE project_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_credential_project ON credential(project_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_template_project ON pipeline_template(project_id);
CREATE INDEX IF NOT EXISTS idx_build_stage_project ON build_stage(project_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_project ON pipeline_snapshot(project_id);
CREATE INDEX IF NOT EXISTS idx_repository_project ON repository(project_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_project ON pipeline_run(project_id);
CREATE INDEX IF NOT EXISTS idx_artifact_project ON artifact(project_id);
CREATE INDEX IF NOT EXISTS idx_application_project ON application(project_id);
CREATE INDEX IF NOT EXISTS idx_deployment_project ON deployment(project_id);
CREATE INDEX IF NOT EXISTS idx_route_project ON route(project_id);
