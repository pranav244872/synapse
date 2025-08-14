-- =============================================
-- Down Migration - Remove Indexes from Teams Table
-- =============================================

-- This script drops the unique index from the 'manager_id' column in the 'teams' table.
-- Using 'IF EXISTS' ensures the script runs without errors even if the index has already been removed.

DROP INDEX IF EXISTS idx_teams_manager_id_unique;
