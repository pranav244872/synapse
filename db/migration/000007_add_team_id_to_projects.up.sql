-- =============================================
-- Migration Up: 000007_add_team_id_and_description_to_projects.up.sql
-- =============================================
-- This migration updates the 'projects' table with:
-- 1. A new 'team_id' column to link each project to a team.
-- 2. A new 'description' column for richer project metadata.
-- 3. A foreign key constraint from 'projects.team_id' to 'teams.id'.

-- Section 1: Add New Columns
-- -------------------------------------------
ALTER TABLE projects
ADD COLUMN team_id BIGINT;

ALTER TABLE projects
ADD COLUMN description TEXT;

-- Section 2: Make 'team_id' NOT NULL
-- -------------------------------------------
-- Assumes you've already backfilled existing projects with team IDs.
ALTER TABLE projects
ALTER COLUMN team_id SET NOT NULL;

-- Section 3: Add Foreign Key Constraint
-- -------------------------------------------
-- Cascades deletes from teams to their projects.
ALTER TABLE projects
ADD CONSTRAINT fk_projects_team
FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
