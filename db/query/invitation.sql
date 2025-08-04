-- SQLC-formatted queries for the "invitations" table.

-- name: CreateInvitation :one
WITH new_invitation AS (
    INSERT INTO invitations (
        email, invitation_token, role_to_invite, inviter_id, expires_at, team_id
    ) VALUES (
        $1, $2, $3, $4, $5, $6
    ) RETURNING *
)
SELECT
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email
FROM
    new_invitation i
LEFT JOIN
    users u ON i.inviter_id = u.id;

-- name: GetInvitationByID :one
-- Retrieves a single invitation by its ID for validation and status checking.
SELECT 
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email
FROM
    invitations i
LEFT JOIN
    users u ON i.inviter_id = u.id
WHERE 
    i.id = $1
LIMIT 1;

-- name: GetInvitationByToken :one
SELECT
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email
FROM
    invitations i
LEFT JOIN
    users u ON i.inviter_id = u.id
WHERE
    i.invitation_token = $1 AND i.status = 'pending' AND i.expires_at > now()
LIMIT 1;

-- name: GetInvitationByEmail :one
SELECT
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email
FROM
    invitations i
LEFT JOIN
    users u ON i.inviter_id = u.id
WHERE
    i.email = $1 AND i.status = 'pending'
LIMIT 1;

-- name: UpdateInvitationStatus :one
WITH updated_invitation AS (
    UPDATE invitations
    SET status = $2
    WHERE invitations.id = $1
    RETURNING *
)
SELECT
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email
FROM
    updated_invitation i
LEFT JOIN
    users u ON i.inviter_id = u.id;

-- name: DeleteInvitation :exec
DELETE FROM invitations
WHERE invitations.id = $1 AND invitations.status = 'pending';

-- ----------------------------------------------------------------
-- Invitation List Queries (Admin & Manager)
-- ----------------------------------------------------------------

-- name: ListAllInvitations :many
SELECT
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email,
    COALESCE(u.role::text, 'unknown') as inviter_role
FROM
    invitations i
LEFT JOIN
    users u ON i.inviter_id = u.id
ORDER BY
    i.created_at DESC
LIMIT $1
OFFSET $2;

-- name: CountAllInvitations :one
SELECT count(*) FROM invitations;

-- name: ListInvitationsByInviter :many
SELECT
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email,
    COALESCE(u.role::text, 'unknown')::text as inviter_role
FROM
    invitations i
LEFT JOIN
    users u ON i.inviter_id = u.id
WHERE
    i.inviter_id = $1
ORDER BY
    i.created_at DESC
LIMIT $2
OFFSET $3;

-- name: CountInvitationsByInviter :one
SELECT count(*) FROM invitations WHERE inviter_id = $1;

-- name: ListInvitationsByInviterRole :many
SELECT
    i.id, i.email, i.invitation_token, i.role_to_invite, i.inviter_id, i.status, i.created_at, i.expires_at, i.team_id,
    COALESCE(u.name, '') as inviter_name,
    COALESCE(u.email, '') as inviter_email,
    COALESCE(u.role, 'unknown')::text as inviter_role
FROM
    invitations i
LEFT JOIN
    users u ON i.inviter_id = u.id
WHERE
    u.role = $1
ORDER BY
    i.created_at DESC
LIMIT $2
OFFSET $3;

-- name: CountInvitationsByInviterRole :one
SELECT count(*)
FROM invitations i
LEFT JOIN
    users u ON i.inviter_id = u.id
WHERE u.role = $1;
