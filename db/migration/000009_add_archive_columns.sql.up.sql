-- =================================================================
-- Migration Up: 000009_add_archive_columns_to_projects_and_tasks.up.sql
-- =================================================================
-- This migration adds soft-delete (archive) functionality to the
-- 'projects' and 'tasks' tables. Archived records will be excluded from
-- normal application queries but retained for use cases such as
-- historical analytics or ML training.

-- Add archive columns to the 'projects' table.
ALTER TABLE projects 
ADD COLUMN archived BOOLEAN DEFAULT false NOT NULL,
ADD COLUMN archived_at TIMESTAMP;

-- Add archive columns to the 'tasks' table.
ALTER TABLE tasks
ADD COLUMN archived BOOLEAN DEFAULT false NOT NULL,
ADD COLUMN archived_at TIMESTAMP;

-- Create indexes to improve performance of archive-related queries.
CREATE INDEX idx_projects_archived ON projects(archived) WHERE archived = true;
CREATE INDEX idx_projects_team_archived ON projects(team_id, archived);
CREATE INDEX idx_tasks_archived ON tasks(archived) WHERE archived = true;
CREATE INDEX idx_tasks_project_archived ON tasks(project_id, archived);

-- Add comments to describe the purpose of the archive columns.
COMMENT ON COLUMN projects.archived IS 'Soft delete flag - archived projects are hidden from normal operations but preserved for ML training';
COMMENT ON COLUMN projects.archived_at IS 'Timestamp when project was archived';
COMMENT ON COLUMN tasks.archived IS 'Soft delete flag - archived tasks are hidden from normal operations but preserved for ML training';
COMMENT ON COLUMN tasks.archived_at IS 'Timestamp when task was archived';

