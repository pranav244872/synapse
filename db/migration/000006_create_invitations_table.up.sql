-- =============================================
-- Migration Up: 000006_create_invitations_table.up.sql
-- =============================================
-- This migration creates the 'invitations' table to manage user invitation flows.
-- 1. Creates the 'invitations' table to store tokenized invitation details.
-- 2. Establishes a foreign key link to the 'users' table for tracking who sent the invitation.

-- Section 1: Create Invitations Table
-- -------------------------------------------
-- This table will track pending, accepted, and expired invitations sent to new users.
CREATE TABLE invitations (
    -- Unique identifier for each invitation record.
    id BIGSERIAL PRIMARY KEY,

    -- The email address of the person being invited.
    email VARCHAR(255) NOT NULL UNIQUE,

    -- A secure, random token sent in the invitation link. Must be unique.
    invitation_token VARCHAR(255) NOT NULL UNIQUE,

    -- The role assigned to the user upon successful registration (e.g., 'manager', 'engineer').
    role_to_invite user_role NOT NULL,

    -- The ID of the user who sent the invitation.
    inviter_id BIGINT NOT NULL,

    -- The current status of the invitation.
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- Values: pending, accepted, expired

    -- Timestamp for when the invitation was created.
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Timestamp for when the invitation token is no longer valid.
    expires_at TIMESTAMP NOT NULL,

    -- A check constraint to ensure the status value is one of the allowed options.
    CONSTRAINT chk_status CHECK (status IN ('pending', 'accepted', 'expired'))
);

-- Section 2: Add Foreign Key Constraint
-- -------------------------------------------
-- Links the 'inviter_id' to the 'users' table to maintain referential integrity.
-- ON DELETE CASCADE ensures that if the inviting user is deleted, their pending
-- invitations are also removed from the database.
ALTER TABLE invitations
ADD CONSTRAINT fk_inviter
FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE CASCADE;
