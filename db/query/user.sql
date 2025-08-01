-- SQLC-formatted queries for the "users" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateUser :one
-- Inserts a new user into the users table.
INSERT INTO users (
    name,
    email,
    team_id,
	password_hash,
	role
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUser :one
-- Retrieves a single user by their unique ID.
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
-- Retrieves a single user by their unique email address.
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: ListUsers :many
-- Retrieves a paginated list of all users, ordered by ID.
SELECT * FROM users
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: ListUsersByTeam :many
-- Retrieves a paginated list of all users belonging to a specific team.
SELECT * FROM users
WHERE team_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateUser :one
-- Updates the details of a specific user.
-- Uses sqlc.narg() to allow for partial updates. Only non-NULL arguments will be used to update fields.
UPDATE users
SET
    name = coalesce(sqlc.narg(name), name),
    team_id = coalesce(sqlc.narg(team_id), team_id),
    availability = coalesce(sqlc.narg(availability), availability),
	role = coalesce(sqlc.narg(role), role)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RemoveUserFromTeam :one
UPDATE users
SET team_id = NULL
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
-- Deletes a user from the database by their ID.
DELETE FROM users
WHERE id = $1;
