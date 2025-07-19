-- SQLC-formatted queries for the "task_required_skills" junction table.
-- These follow the conventions for use with the sqlc tool.

-- name: AddSkillToTask :one
-- Adds a required skill to a specific task.
INSERT INTO task_required_skills (
    task_id,
    skill_id
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetSkillsForTask :many
-- Retrieves all skills required for a specific task by joining with the skills table.
SELECT s.* FROM skills s
JOIN task_required_skills trs ON s.id = trs.skill_id
WHERE trs.task_id = $1;

-- name: GetTasksForSkill :many
-- Retrieves all tasks that require a specific skill by joining with the tasks table.
SELECT t.* FROM tasks t
JOIN task_required_skills trs ON t.id = trs.task_id
WHERE trs.skill_id = $1;

-- name: RemoveSkillFromTask :exec
-- Removes a required skill from a specific task.
DELETE FROM task_required_skills
WHERE task_id = $1 AND skill_id = $2;
