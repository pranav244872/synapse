-- SQLC-formatted queries for the "projects" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateProject :one
-- Inserts a new project into the projects table.
INSERT INTO projects (
    project_name,
    team_id,
    description
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetProject :one
-- Retrieves a single project by its unique ID.
SELECT * FROM projects
WHERE id = $1
LIMIT 1;

-- name: GetProjectByIDAndTeam :one
-- Retrieves a project only if it belongs to the specified team.
SELECT * FROM projects
WHERE id = $1 AND team_id = $2
LIMIT 1;

-- name: ListProjects :many
-- Retrieves a paginated list of all projects, ordered by ID.
SELECT * FROM projects
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: ListProjectsByTeam :many
-- Lists all projects for a specific team.
SELECT * FROM projects
WHERE team_id = $1
ORDER BY id
LIMIT $2 OFFSET $3;

-- name: UpdateProject :one
-- Updates the project's name and/or description.
-- If a value is null, the existing value will be retained.
UPDATE projects
SET
    project_name = COALESCE($3, project_name),
    description = COALESCE($4, description)
WHERE id = $1 AND team_id = $2
RETURNING *;

-- name: DeleteProject :exec
-- Deletes a project from the database by its ID.
DELETE FROM projects
WHERE id = $1;

-- name: DeleteProjectByTeam :exec
-- Deletes a project that belongs to a specific team.
DELETE FROM projects
WHERE id = $1 AND team_id = $2;

-- name: CountProjects :one
SELECT count(*) FROM projects;
