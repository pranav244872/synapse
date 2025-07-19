-- SQLC-formatted queries for the "tasks" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateTask :one
-- Inserts a new task into the tasks table.
INSERT INTO tasks (
    project_id,
    title,
    description,
    status,
    priority,
    assignee_id
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetTask :one
-- Retrieves a single task by its unique ID.
SELECT * FROM tasks
WHERE id = $1 LIMIT 1;

-- name: ListTasks :many
-- Retrieves a paginated list of all tasks, ordered by creation date.
SELECT * FROM tasks
ORDER BY created_at DESC
LIMIT $1
OFFSET $2;

-- name: ListTasksByProject :many
-- Retrieves a paginated list of all tasks for a given project.
SELECT * FROM tasks
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: ListTasksByAssignee :many
-- Retrieves a paginated list of all tasks assigned to a specific user.
SELECT * FROM tasks
WHERE assignee_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateTask :one
-- Updates the details of a specific task.
-- Uses sqlc.narg() to allow for partial updates of any field.
UPDATE tasks
SET
    project_id = COALESCE(sqlc.narg(project_id), project_id),
    title = COALESCE(sqlc.narg(title), title),
    description = COALESCE(sqlc.narg(description), description),
    status = COALESCE(sqlc.narg(status), status),
    priority = COALESCE(sqlc.narg(priority), priority),
    assignee_id = COALESCE(sqlc.narg(assignee_id), assignee_id),
    completed_at = COALESCE(sqlc.narg(completed_at), completed_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteTask :exec
-- Deletes a task from the database by its ID.
DELETE FROM tasks
WHERE id = $1;
