-- =============================================
-- Migration Up: 000005_add_admin_role_and_team_manager.up.sql
-- =============================================
-- This migration introduces a hierarchical role system and links teams to a specific manager.
-- 1. Adds a new 'admin' role to the user_role enum for superuser capabilities.
-- 2. Adds a 'manager_id' column to the 'teams' table to designate a team manager.

-- Section 1: Enhance User Roles
-- -------------------------------------------
-- Add the 'admin' value to the existing user_role ENUM.
-- This role is intended for superusers who can manage teams and invite managers.
ALTER TYPE user_role ADD VALUE 'admin';

-- Section 2: Update Teams Table for Manager Assignment
-- -------------------------------------------
-- Add a unique 'manager_id' column to the 'teams' table.
-- This creates a clear, one-to-one relationship between a team and its manager.
ALTER TABLE teams
ADD COLUMN manager_id BIGINT UNIQUE;

-- Add a foreign key constraint to link 'manager_id' to the 'users' table.
-- ON DELETE SET NULL ensures that if a manager's user account is deleted,
-- the team becomes managerless instead of being deleted itself.
ALTER TABLE teams
ADD CONSTRAINT fk_manager
FOREIGN KEY (manager_id) REFERENCES users(id) ON DELETE SET NULL;
