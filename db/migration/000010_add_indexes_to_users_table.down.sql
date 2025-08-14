-- =============================================
-- Drop Script (Down)
-- =============================================
-- This script removes the indexes added to the `users` table for performance optimization.
-- It is written to be safe and idempotent using 'IF EXISTS' clauses.
-- Note: The pg_trgm extension is *not* removed here, as it may be used elsewhere.

-- Section 6: Drop Indexes for Performance Optimization
-- -------------------------------------------
-- Cleanly remove all custom indexes created on the `users` table.

DROP INDEX IF EXISTS idx_users_email_gin;
DROP INDEX IF EXISTS idx_users_name_gin;
DROP INDEX IF EXISTS idx_users_team_id_role_name;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_team_id;
DROP INDEX IF EXISTS idx_users_email_unique;

-- Note: The pg_trgm extension is NOT dropped here, as it may be in use by other tables
-- or parts of the application. It should be dropped separately if it is truly no longer needed.
-- To drop it, you would run: DROP EXTENSION IF EXISTS pg_trgm;
