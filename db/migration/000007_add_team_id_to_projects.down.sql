-- =============================================
-- Migration Down: 000007_add_team_id_and_description_to_projects.down.sql
-- =============================================
-- This reverts the schema changes to the 'projects' table by:
-- 1. Dropping the foreign key constraint.
-- 2. Removing the 'team_id' and 'description' columns.

-- Section 1: Drop Foreign Key Constraint
ALTER TABLE projects
DROP CONSTRAINT IF EXISTS fk_projects_team;

-- Section 2: Drop Columns
ALTER TABLE projects
DROP COLUMN IF EXISTS team_id;

ALTER TABLE projects
DROP COLUMN IF EXISTS description;
