// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: task.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createTask = `-- name: CreateTask :one

INSERT INTO tasks (
    project_id,
    title,
    description,
    status,
    priority,
    assignee_id
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, project_id, title, description, status, priority, assignee_id, created_at, completed_at
`

type CreateTaskParams struct {
	ProjectID   pgtype.Int8
	Title       string
	Description pgtype.Text
	Status      TaskStatus
	Priority    TaskPriority
	AssigneeID  pgtype.Int8
}

// SQLC-formatted queries for the "tasks" table.
// These follow the conventions for use with the sqlc tool.
// Inserts a new task into the tasks table.
func (q *Queries) CreateTask(ctx context.Context, arg CreateTaskParams) (Task, error) {
	row := q.db.QueryRow(ctx, createTask,
		arg.ProjectID,
		arg.Title,
		arg.Description,
		arg.Status,
		arg.Priority,
		arg.AssigneeID,
	)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.ProjectID,
		&i.Title,
		&i.Description,
		&i.Status,
		&i.Priority,
		&i.AssigneeID,
		&i.CreatedAt,
		&i.CompletedAt,
	)
	return i, err
}

const deleteTask = `-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1
`

// Deletes a task from the database by its ID.
func (q *Queries) DeleteTask(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, deleteTask, id)
	return err
}

const getTask = `-- name: GetTask :one
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at FROM tasks
WHERE id = $1 LIMIT 1
`

// Retrieves a single task by its unique ID.
func (q *Queries) GetTask(ctx context.Context, id int64) (Task, error) {
	row := q.db.QueryRow(ctx, getTask, id)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.ProjectID,
		&i.Title,
		&i.Description,
		&i.Status,
		&i.Priority,
		&i.AssigneeID,
		&i.CreatedAt,
		&i.CompletedAt,
	)
	return i, err
}

const listTasks = `-- name: ListTasks :many
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at FROM tasks
ORDER BY created_at DESC
LIMIT $1
OFFSET $2
`

type ListTasksParams struct {
	Limit  int32
	Offset int32
}

// Retrieves a paginated list of all tasks, ordered by creation date.
func (q *Queries) ListTasks(ctx context.Context, arg ListTasksParams) ([]Task, error) {
	rows, err := q.db.Query(ctx, listTasks, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Task
	for rows.Next() {
		var i Task
		if err := rows.Scan(
			&i.ID,
			&i.ProjectID,
			&i.Title,
			&i.Description,
			&i.Status,
			&i.Priority,
			&i.AssigneeID,
			&i.CreatedAt,
			&i.CompletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTasksByAssignee = `-- name: ListTasksByAssignee :many
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at FROM tasks
WHERE assignee_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3
`

type ListTasksByAssigneeParams struct {
	AssigneeID pgtype.Int8
	Limit      int32
	Offset     int32
}

// Retrieves a paginated list of all tasks assigned to a specific user.
func (q *Queries) ListTasksByAssignee(ctx context.Context, arg ListTasksByAssigneeParams) ([]Task, error) {
	rows, err := q.db.Query(ctx, listTasksByAssignee, arg.AssigneeID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Task
	for rows.Next() {
		var i Task
		if err := rows.Scan(
			&i.ID,
			&i.ProjectID,
			&i.Title,
			&i.Description,
			&i.Status,
			&i.Priority,
			&i.AssigneeID,
			&i.CreatedAt,
			&i.CompletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTasksByProject = `-- name: ListTasksByProject :many
SELECT id, project_id, title, description, status, priority, assignee_id, created_at, completed_at FROM tasks
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3
`

type ListTasksByProjectParams struct {
	ProjectID pgtype.Int8
	Limit     int32
	Offset    int32
}

// Retrieves a paginated list of all tasks for a given project.
func (q *Queries) ListTasksByProject(ctx context.Context, arg ListTasksByProjectParams) ([]Task, error) {
	rows, err := q.db.Query(ctx, listTasksByProject, arg.ProjectID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Task
	for rows.Next() {
		var i Task
		if err := rows.Scan(
			&i.ID,
			&i.ProjectID,
			&i.Title,
			&i.Description,
			&i.Status,
			&i.Priority,
			&i.AssigneeID,
			&i.CreatedAt,
			&i.CompletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateTask = `-- name: UpdateTask :one
UPDATE tasks
SET
    project_id = COALESCE($1, project_id),
    title = COALESCE($2, title),
    description = COALESCE($3, description),
    status = COALESCE($4, status),
    priority = COALESCE($5, priority),
    assignee_id = COALESCE($6, assignee_id),
    completed_at = COALESCE($7, completed_at)
WHERE id = $8
RETURNING id, project_id, title, description, status, priority, assignee_id, created_at, completed_at
`

type UpdateTaskParams struct {
	ProjectID   pgtype.Int8
	Title       pgtype.Text
	Description pgtype.Text
	Status      NullTaskStatus
	Priority    NullTaskPriority
	AssigneeID  pgtype.Int8
	CompletedAt pgtype.Timestamp
	ID          int64
}

// Updates the details of a specific task.
// Uses sqlc.narg() to allow for partial updates of any field.
func (q *Queries) UpdateTask(ctx context.Context, arg UpdateTaskParams) (Task, error) {
	row := q.db.QueryRow(ctx, updateTask,
		arg.ProjectID,
		arg.Title,
		arg.Description,
		arg.Status,
		arg.Priority,
		arg.AssigneeID,
		arg.CompletedAt,
		arg.ID,
	)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.ProjectID,
		&i.Title,
		&i.Description,
		&i.Status,
		&i.Priority,
		&i.AssigneeID,
		&i.CreatedAt,
		&i.CompletedAt,
	)
	return i, err
}
