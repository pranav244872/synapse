-- ===================================================================
-- Migration Down: 000009_add_archive_columns_to_projects_and_tasks.down.sql
-- ===================================================================
-- This migration reverts the archive functionality added to the
-- 'projects' and 'tasks' tables by removing the archive-related

-- Drop archive columns from the 'projects' table.
ALTER TABLE projects
DROP COLUMN IF EXISTS archived,
DROP COLUMN IF EXISTS archived_at;

-- Drop archive columns from the 'tasks' table.
ALTER TABLE tasks
DROP COLUMN IF EXISTS archived,
DROP COLUMN IF EXISTS archived_at;
