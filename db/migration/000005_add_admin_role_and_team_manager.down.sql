-- =============================================
-- Migration Down: 000005_add_admin_role_and_team_manager.down.sql
-- =============================================
-- This migration reverts the team management changes by removing the 'manager_id'
-- column and its foreign key constraint from the 'teams' table.

-- Section 1: Revert Teams Table Changes
-- -------------------------------------------
-- The foreign key constraint must be dropped before the column can be removed.
-- Using IF EXISTS prevents errors if the script is run more than once.
ALTER TABLE teams
DROP CONSTRAINT IF EXISTS fk_manager;

-- Drop the manager_id column from the teams table.
ALTER TABLE teams
DROP COLUMN IF EXISTS manager_id;

-- Section 2: Revert User Role Type Enhancement
-- -------------------------------------------
-- IMPORTANT: PostgreSQL does not support 'DROP VALUE' for ENUM types.
-- Reverting the 'ALTER TYPE ... ADD VALUE' command is a complex and potentially
-- destructive operation that involves creating a new type and migrating all
-- existing data.
--
-- Therefore, this part of the 'down' migration is intentionally left blank.
-- The 'admin' value will remain in the user_role type definition but can be
-- ignored by the application logic.
