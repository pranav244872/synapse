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

-- name: UpdateProject :one
-- Updates a project's name and description.
UPDATE projects
SET project_name = $3,
    description = $4
WHERE id = $1 AND team_id = $2
RETURNING *;

-- name: DeleteProject :exec
-- Deletes a project from the database by its ID.
DELETE FROM projects
WHERE id = $1;

-- Archive a single active project by ID and team, returning its details
-- name: ArchiveProject :one
UPDATE projects 
SET archived = true, archived_at = now()
WHERE id = $1 AND team_id = $2 AND archived = false
RETURNING id, project_name, team_id, description, archived, archived_at;

-- Unarchive a single archived project by ID and team, returning its details
-- name: UnarchiveProject :one  
UPDATE projects
SET archived = false, archived_at = NULL
WHERE id = $1 AND team_id = $2 AND archived = true
RETURNING id, project_name, team_id, description, archived, archived_at;

-- List paginated active (non-archived) projects for a team
-- name: ListActiveProjectsByTeam :many
SELECT id, project_name, team_id, description, archived, archived_at
FROM projects 
WHERE team_id = $1 AND archived = false
ORDER BY id
LIMIT $2 OFFSET $3;

-- List paginated archived projects for a team, sorted by archive date
-- name: ListArchivedProjectsByTeam :many
SELECT id, project_name, team_id, description, archived, archived_at
FROM projects
WHERE team_id = $1 AND archived = true  
ORDER BY archived_at DESC
LIMIT $2 OFFSET $3;

-- Count the number of active (non-archived) projects in a team
-- name: CountActiveProjectsByTeam :one
SELECT count(*) FROM projects 
WHERE team_id = $1 AND archived = false;

-- Count the number of archived projects in a team
-- name: CountArchivedProjectsByTeam :one
SELECT count(*) FROM projects
WHERE team_id = $1 AND archived = true;

-- List paginated active projects for a team
-- name: ListProjectsByTeam :many
SELECT id, project_name, team_id, description, archived, archived_at FROM projects
WHERE team_id = $1 AND archived = false
ORDER BY id
LIMIT $2 OFFSET $3;

-- Count only active projects for a team
-- name: CountProjectsByTeam :one
SELECT count(*) FROM projects WHERE team_id = $1 AND archived = false;
