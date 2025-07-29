-- =============================================
-- Migration Down: 000004_add_user_auth_and_roles.down.sql
-- =============================================
-- This migration reverts the changes made in the corresponding 'up' script.
-- It removes the 'password_hash' and 'role' columns from the 'users' table and
-- drops the 'user_role' ENUM type.

-- Section 1: Revert Users Table Changes
-- -------------------------------------------
-- Remove the 'password_hash' and 'role' columns from the 'users' table.
-- This must be done before dropping the 'user_role' type to remove the dependency.
ALTER TABLE users
DROP COLUMN password_hash,
DROP COLUMN role;

-- Section 2: Remove User Role Type
-- -------------------------------------------
-- Drop the 'user_role' ENUM type as it is no longer in use by any column.
DROP TYPE user_role;
