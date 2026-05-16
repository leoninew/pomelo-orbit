-- v0.7.4: Add is_active column to project table

ALTER TABLE project ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT 1;
