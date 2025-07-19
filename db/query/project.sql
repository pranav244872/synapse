-- SQLC-formatted queries for the "projects" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateProject :one
-- Inserts a new project into the projects table.
INSERT INTO projects (
    project_name
) VALUES (
    $1
) RETURNING *;

-- name: GetProject :one
-- Retrieves a single project by its unique ID.
SELECT * FROM projects
WHERE id = $1 LIMIT 1;

-- name: ListProjects :many
-- Retrieves a paginated list of all projects, ordered by ID.
SELECT * FROM projects
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateProject :one
-- Updates the name of a specific project.
UPDATE projects
SET project_name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
-- Deletes a project from the database by its ID.
DELETE FROM projects
WHERE id = $1;
