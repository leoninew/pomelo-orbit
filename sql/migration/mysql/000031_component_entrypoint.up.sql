ALTER TABLE version_component
    ADD COLUMN entrypoint_json TEXT NULL;

UPDATE version_component
SET entrypoint_json = '[]'
WHERE entrypoint_json IS NULL;

ALTER TABLE version_component
    MODIFY COLUMN entrypoint_json TEXT NOT NULL;
