-- =============================================
-- Up Migration - Add Indexes for Tasks & Related Tables
-- =============================================
-- This script adds several indexes to optimize queries related to tasks,
-- including partial, composite, and GIN indexes.

-- Enable the pg_trgm extension for efficient text searching with ILIKE.
-- This is crucial for the GetEngineerTaskHistory query.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ====== Indexes on the "tasks" table ======

-- Optimizes listing/counting active tasks for a project, sorted by creation date.
-- This partial index is small and fast because it only includes non-archived tasks.
-- Covers: ListActiveTasksByProject, CountActiveTasksByProject, ListTasksWithAssigneeNames
CREATE INDEX IF NOT EXISTS idx_tasks_project_id_created_at_active ON "tasks" (project_id, created_at DESC) WHERE archived = false;

-- Optimizes listing/counting archived tasks for a project, sorted by archive date.
-- Covers: ListArchivedTasksByProject, CountArchivedTasksByProject
CREATE INDEX IF NOT EXISTS idx_tasks_project_id_archived_at_archived ON "tasks" (project_id, archived_at DESC) WHERE archived = true;

-- Optimizes fetching tasks for a specific user based on status (e.g., 'in_progress').
-- Covers: ListTasksByAssignee, GetCurrentTaskForEngineer
CREATE INDEX IF NOT EXISTS idx_tasks_assignee_id_status_active ON "tasks" (assignee_id, status) WHERE archived = false;

-- Highly specific index for fetching a user's completed task history, sorted by completion.
-- Covers: GetEngineerTaskHistory, GetEngineerTaskHistoryCount
CREATE INDEX IF NOT EXISTS idx_tasks_assignee_id_completed_at_done ON "tasks" (assignee_id, completed_at DESC) WHERE status = 'done' AND archived = false;

-- GIN index for fast, case-insensitive text search on the task title.
-- Covers: GetEngineerTaskHistory, GetEngineerTaskHistoryCount
CREATE INDEX IF NOT EXISTS idx_tasks_title_gin ON "tasks" USING GIN (title gin_trgm_ops);

-- ====== Indexes on Junction Tables ======

-- The primary key on user_skills is (user_id, skill_id). This new index
-- is needed to efficiently find all users who have a specific skill.
CREATE INDEX IF NOT EXISTS idx_user_skills_skill_id ON "user_skills" (skill_id);

-- The primary key on task_required_skills is (task_id, skill_id). This new index
-- is needed to efficiently find all tasks that require a specific skill.
CREATE INDEX IF NOT EXISTS idx_task_required_skills_skill_id ON "task_required_skills" (skill_id);
