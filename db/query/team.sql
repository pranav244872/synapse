-- SQLC-formatted queries for the "teams" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateTeam :one
-- Inserts a new team into the teams table.
INSERT INTO teams (
    team_name
) VALUES (
    $1
) RETURNING *;

-- name: GetTeam :one
-- Retrieves a single team by its unique ID.
SELECT * FROM teams
WHERE id = $1 LIMIT 1;

-- name: ListTeams :many
-- Retrieves a paginated list of all teams, ordered by ID.
SELECT * FROM teams
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateTeam :one
-- Updates the name of a specific team.
UPDATE teams
SET team_name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteTeam :exec
-- Deletes a team from the database by its ID.
DELETE FROM teams
WHERE id = $1;
