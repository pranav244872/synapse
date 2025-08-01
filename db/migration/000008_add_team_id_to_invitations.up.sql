-- =================================================================
-- Migration Up: 000008_add_team_id_to_invitations.up.sql
-- =================================================================
-- This migration adds a 'team_id' column to the 'invitations' table.
-- This is crucial for linking an invitation directly to a team, which is
-- mandatory for inviting managers and assigning engineers to their manager's team.

-- Add the team_id column, making it nullable as not all roles might require it initially,
-- though application logic will enforce it.
ALTER TABLE invitations
ADD COLUMN team_id BIGINT;

-- Add a foreign key constraint to link 'team_id' to the 'teams' table.
-- ON DELETE SET NULL means if a team is deleted, the team_id on any pending
-- invitations will be set to NULL. This prevents accidental deletion of invitations
-- and can be handled by application logic (e.g., invalidate the invitation).
ALTER TABLE invitations
ADD CONSTRAINT fk_invitations_team
FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL;

-- Add an index on the new team_id column for faster lookups.
CREATE INDEX idx_invitations_team_id ON invitations(team_id);
