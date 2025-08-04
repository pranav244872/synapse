-- SQLC-formatted queries for the "teams" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CountTeams :one
-- Returns the total number of teams.
SELECT count(*) FROM teams;

-- name: CreateTeam :one
-- Inserts a new team into the teams table.
-- Added manager_id to allow assigning a manager upon creation.
-- The manager_id can be NULL if a team is created without an immediate manager.
INSERT INTO teams (
  team_name,
  manager_id
) VALUES (
  $1, $2
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
-- Updates the name and/or the manager of a specific team.
-- Uses COALESCE and sqlc.narg to allow optional updates.
-- You can update the name, the manager_id, or both in a single call.
UPDATE teams
SET 
  team_name = COALESCE(sqlc.narg(team_name), team_name),
  manager_id = sqlc.narg(manager_id)
WHERE id = $1
RETURNING *;

-- name: DeleteTeam :exec
-- Deletes a team from the database by its ID.
DELETE FROM teams
WHERE id = $1;

-- Get a team by the manager's user ID.
-- Since manager_id is unique, this will return at most one team.
-- name: GetTeamByManagerID :one
SELECT * FROM teams
WHERE manager_id = $1 LIMIT 1;

-- List all teams and include their manager's details.
-- This uses a LEFT JOIN to ensure teams without a manager are still included.
-- This is useful for UI displays to avoid separate lookups for manager names.
-- name: ListTeamsWithManagers :many
SELECT *
FROM teams t
LEFT JOIN users u ON t.manager_id = u.id
ORDER BY t.id
LIMIT $1
OFFSET $2;

-- name: ListUnmanagedTeams :many
-- Retrieves all teams that do not have a manager_id assigned.
-- Useful for populating dropdowns for manager assignment.
SELECT * FROM teams
WHERE manager_id IS NULL
ORDER BY team_name;

-- name: SetTeamManager :one
-- Sets the manager for a specific team.
UPDATE teams
SET manager_id = $2
WHERE id = $1
RETURNING *;
