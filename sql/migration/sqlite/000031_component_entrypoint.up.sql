ALTER TABLE version_component
    ADD COLUMN entrypoint_json TEXT NOT NULL DEFAULT '[]';
