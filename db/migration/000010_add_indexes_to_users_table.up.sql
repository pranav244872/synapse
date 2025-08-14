-- =============================================
-- Create Script (Up)
-- =============================================
-- This script creates all necessary indexes to optimize query performance,
-- particularly around user search, filtering by team or role, and enforcing uniqueness.

-- Indexes for Performance Optimization
-- -------------------------------------------
-- These indexes are designed to improve performance of key application queries,
-- such as user search, team-based filtering, and email lookups.

-- Enable the pg_trgm extension for efficient text searching with LIKE/ILIKE.
-- This is crucial for the SearchUsers and CountSearchUsers queries.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Index for fast, case-insensitive lookup by email.
-- Crucially supports GetUserByEmail and enforces email uniqueness.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users (LOWER(email));

-- Index on the team_id foreign key.
-- Speeds up all queries that filter or join by team, such as ListUsersByTeam,
-- SearchUsers, and ListEngineersByTeam.
CREATE INDEX IF NOT EXISTS idx_users_team_id ON users (team_id);

-- Index on the role column.
-- Optimizes filtering by role in queries like CountSearchUsers and ListEngineersByTeam.
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

-- Composite index to optimize listing engineers in a team, sorted by name.
-- Covers the exact pattern for the ListEngineersByTeam query.
CREATE INDEX IF NOT EXISTS idx_users_team_id_role_name ON users (team_id, role, name);

-- GIN index for fast, case-insensitive substring searches on the user's name.
-- Dramatically improves performance of the LIKE clause in SearchUsers.
CREATE INDEX IF NOT EXISTS idx_users_name_gin ON users USING GIN (LOWER(name) gin_trgm_ops);

-- GIN index for fast, case-insensitive substring searches on the user's email.
-- Dramatically improves performance of the LIKE clause in SearchUsers.
CREATE INDEX IF NOT EXISTS idx_users_email_gin ON users USING GIN (LOWER(email) gin_trgm_ops);
