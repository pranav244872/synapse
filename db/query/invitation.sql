-- SQLC-formatted queries for the "invitations" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateInvitation :one
-- Inserts a new invitation record into the database, including the team association.
INSERT INTO invitations (
    email,
    invitation_token,
    role_to_invite,
    inviter_id,
    expires_at,
    team_id
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetInvitationByToken :one
-- Retrieves a single, non-expired invitation by its unique token.
-- The user will provide this token when they sign up.
SELECT * FROM invitations
WHERE
    invitation_token = $1
    AND status = 'pending'
    AND expires_at > now()
LIMIT 1;

-- name: GetInvitationByEmail :one
-- Retrieves a pending invitation for a given email to prevent sending duplicate invites.
SELECT * FROM invitations
WHERE
    email = $1
    AND status = 'pending'
LIMIT 1;

-- name: UpdateInvitationStatus :one
-- Updates the status of an invitation, for example, to 'accepted' after the user successfully registers.
UPDATE invitations
SET status = $2
WHERE id = $1
RETURNING *;
