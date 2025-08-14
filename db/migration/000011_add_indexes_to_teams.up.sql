-- =============================================
-- Up Migration - Add Indexes to Teams Table
-- =============================================

-- This script adds a unique index to the 'manager_id' column in the 'teams' table.

-- A unique index on 'manager_id' is created to serve two purposes:
-- 1.  **Performance**: It dramatically speeds up lookups based on the manager's ID,
--     which is required by the 'GetTeamByManagerID' and 'ListTeamsWithManagers' queries.
-- 2.  **Data Integrity**: It enforces the business rule that a user can be the manager of at most one team.
-- Note: PostgreSQL's unique index allows multiple NULL values, which correctly models
-- the scenario of having multiple teams without a manager.

CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_manager_id_unique ON "teams" (manager_id);
