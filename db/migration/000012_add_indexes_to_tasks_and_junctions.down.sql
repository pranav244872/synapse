-- =============================================
-- Down Migration - Remove Indexes for Tasks & Related Tables
-- =============================================
-- This script removes all indexes created in the corresponding 'up' migration.
-- The order is the reverse of creation.

DROP INDEX IF EXISTS idx_task_required_skills_skill_id;
DROP INDEX IF EXISTS idx_user_skills_skill_id;

DROP INDEX IF EXISTS idx_tasks_title_gin;
DROP INDEX IF EXISTS idx_tasks_assignee_id_completed_at_done;
DROP INDEX IF EXISTS idx_tasks_assignee_id_status_active;
DROP INDEX IF EXISTS idx_tasks_project_id_archived_at_archived;
DROP INDEX IF EXISTS idx_tasks_project_id_created_at_active;

-- Note: The pg_trgm extension is NOT dropped here, as it may be in use by other tables.
-- To drop it, you would run: DROP EXTENSION IF EXISTS pg_trgm;
