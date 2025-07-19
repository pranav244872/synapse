-- SQLC-formatted queries for the "skills" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateSkill :one
-- Inserts a new skill into the skills table.
INSERT INTO skills (
    skill_name
) VALUES (
    $1
) RETURNING *;

-- name: GetSkill :one
-- Retrieves a single skill by its unique ID.
SELECT * FROM skills
WHERE id = $1 LIMIT 1;

-- name: ListSkills :many
-- Retrieves a paginated list of all skills, ordered by ID.
SELECT * FROM skills
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateSkill :one
-- Updates the name of a specific skill.
-- Uses sqlc.narg() to allow for partial updates.
UPDATE skills
SET skill_name = $2
WHERE id = $1
RETURNING *;

-- name: DeleteSkill :exec
-- Deletes a skill from the database by its ID.
DELETE FROM skills
WHERE id = $1;
