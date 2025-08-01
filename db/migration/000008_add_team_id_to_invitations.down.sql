-- Migration Down: 000008_add_team_id_to_invitations.down.sql
-- ===================================================================
-- This migration reverts the changes made in the UP script.
-- It removes the foreign key constraint, index, and the 'team_id'
-- column from the 'invitations' table.

-- Drop the foreign key constraint if it exists.
ALTER TABLE invitations
DROP CONSTRAINT IF EXISTS fk_invitations_team;

-- Drop the index on the team_id column, if it exists.
DROP INDEX IF EXISTS idx_invitations_team_id;

-- Drop the team_id column itself.
ALTER TABLE invitations
DROP COLUMN IF EXISTS team_id;
