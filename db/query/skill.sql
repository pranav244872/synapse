-- SQLC-formatted queries for the "skills" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateSkill :one
-- Inserts a new skill into the skills table.
INSERT INTO skills (
    skill_name,
    is_verified
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetSkill :one
-- Retrieves a single skill by its unique ID.
SELECT * FROM skills
WHERE id = $1
LIMIT 1;

-- name: ListSkills :many
-- Retrieves a paginated list of all skills, ordered by ID.
SELECT * FROM skills
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateSkill :one
-- Updates the name of a specific skill.
UPDATE skills
SET skill_name = $2
WHERE id = $1
RETURNING *;

-- name: UpsertSkill :one
INSERT INTO skills (skill_name, is_verified)
VALUES ($1, $2)
ON CONFLICT (skill_name) 
DO UPDATE SET 
  skill_name = EXCLUDED.skill_name -- This is a no-op but allows RETURNING to work for existing rows
RETURNING *;

-- name: DeleteSkill :exec
-- Deletes a skill from the database by its ID.
DELETE FROM skills
WHERE id = $1;

-- name: GetSkillByName :one
-- Retrieves a skill by its name (case-insensitive).
SELECT * FROM skills
WHERE LOWER(skill_name) = LOWER($1)
LIMIT 1;

-- name: UpdateSkillVerification :one
-- Updates the is_verified status of a skill.
UPDATE skills
SET is_verified = $2
WHERE id = $1
RETURNING *;

-- name: ListSkillsByNames :many
SELECT * FROM skills
WHERE skill_name = ANY($1::text[]);

-- name: CreateManySkills :many
INSERT INTO skills (skill_name, is_verified)
SELECT unnest($1::text[]), unnest($2::boolean[])
RETURNING *;

-- name: ListSkillsByStatus :many
-- Retrieves a paginated list of skills based on their verification status.
SELECT * FROM skills
WHERE is_verified = $1
ORDER BY skill_name
LIMIT $2
OFFSET $3;

-- name: CountSkillsByStatus :one
-- Returns the total number of skills for a given verification status.
SELECT count(*) FROM skills
WHERE is_verified = $1;

-- Retrieves a paginated list of skills filtered by verification status and partial skill name match
-- name: SearchSkillsByStatus :many
SELECT * FROM skills 
WHERE is_verified = $1 
AND LOWER(skill_name) LIKE LOWER($2)
ORDER BY skill_name ASC
LIMIT $3 OFFSET $4;

-- Counts the total number of skills matching a given verification status and partial skill name
-- name: CountSearchSkillsByStatus :one
SELECT count(*) FROM skills 
WHERE is_verified = $1 
AND LOWER(skill_name) LIKE LOWER($2);
