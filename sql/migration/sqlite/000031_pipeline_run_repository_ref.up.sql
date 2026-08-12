-- Replace the pipeline run source ref name with the repository-scoped name.
ALTER TABLE pipeline_run RENAME COLUMN trigger_ref TO repository_ref;
