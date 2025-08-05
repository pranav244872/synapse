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

-- Archive a single active task by ID and return its details
-- name: ArchiveTask :one
UPDATE tasks
SET archived = true, archived_at = now()  
WHERE id = $1 AND archived = false
RETURNING id, project_id, title, description, status, priority, assignee_id, created_at, completed_at, archived, archived_at;

-- Unarchive a single archived task by ID and return its details
-- name: UnarchiveTask :one
UPDATE tasks  
SET archived = false, archived_at = NULL
WHERE id = $1 AND archived = true
RETURNING id, project_id, title, description, status, priority, assignee_id, created_at, completed_at, archived, archived_at;

-- List paginated active (non-archived) tasks for a project, sorted by creation date
-- name: ListActiveTasksByProject :many
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at, archived, archived_at
FROM tasks
WHERE project_id = $1 AND archived = false
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- List paginated archived tasks for a project, sorted by archive date
-- name: ListArchivedTasksByProject :many  
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at, archived, archived_at
FROM tasks
WHERE project_id = $1 AND archived = true
ORDER BY archived_at DESC  
LIMIT $2 OFFSET $3;

-- Count the number of active (non-archived) tasks in a project
-- name: CountActiveTasksByProject :one
SELECT count(*) FROM tasks
WHERE project_id = $1 AND archived = false;

-- Count the number of archived tasks in a project
-- name: CountArchivedTasksByProject :one
SELECT count(*) FROM tasks  
WHERE project_id = $1 AND archived = true;

-- Archive all completed tasks in a project that are not already archived
-- name: ArchiveCompletedTasksByProject :exec
UPDATE tasks
SET archived = true, archived_at = now()
WHERE project_id = $1 AND status = 'done' AND archived = false;

-- List paginated active tasks for a project (updated version)
-- name: ListTasksByProject :many
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at, archived, archived_at FROM tasks
WHERE project_id = $1 AND archived = false
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- List paginated active tasks assigned to a specific user
-- name: ListTasksByAssignee :many
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at, archived, archived_at FROM tasks
WHERE assignee_id = $1 AND archived = false
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- Count the number of active (non-archived) tasks in a project with a specific status
-- name: CountTasksByProjectAndStatus :one
SELECT count(*) FROM tasks 
WHERE project_id = $1 AND status = $2 AND archived = false;

-- List tasks in a project along with assignee names, with pagination and sorted by newest first
-- name: ListTasksWithAssigneeNames :many
SELECT t.id, t.title, t.status, t.priority, t.assignee_id, 
       u.name as assignee_name
FROM tasks t
LEFT JOIN users u ON t.assignee_id = u.id
WHERE t.project_id = $1 AND t.archived = false
ORDER BY t.created_at DESC
LIMIT $2 OFFSET $3;
