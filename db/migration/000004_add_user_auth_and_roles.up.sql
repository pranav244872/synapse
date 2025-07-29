-- =============================================
-- Migration Up: 000004_add_user_auth_and_roles.up.sql
-- =============================================
-- This migration enhances the 'users' table by adding authentication and authorization features.
-- It introduces a 'password_hash' for secure password storage and a 'role' column
-- to differentiate between user types ('manager', 'engineer').

-- Section 1: Define User Role Type
-- -------------------------------------------
-- Create a new ENUM type 'user_role' to enforce valid role values at the database level.
CREATE TYPE user_role AS ENUM ('manager', 'engineer');

-- Section 2: Enhance Users Table
-- -------------------------------------------
-- Add the 'password_hash' and 'role' columns to the 'users' table.
-- 'password_hash' is non-nullable to ensure every user has a password.
-- 'role' is non-nullable and defaults to 'engineer' for new user accounts.
ALTER TABLE users
ADD COLUMN password_hash VARCHAR(255) NOT NULL,
ADD COLUMN role user_role NOT NULL DEFAULT 'engineer';
