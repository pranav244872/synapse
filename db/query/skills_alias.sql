-- SQLC-formatted queries for the "skill_aliases" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateSkillAlias :one
-- Inserts a new skill alias and associates it with a canonical skill.
INSERT INTO skill_aliases (
    alias_name,
    skill_id
) VALUES (
    $1, $2
) RETURNING *;

-- name: GetSkillAlias :one
-- Retrieves a single skill alias by its name (primary key).
SELECT * FROM skill_aliases
WHERE alias_name = $1 LIMIT 1;

-- name: ListSkillAliases :many
-- Retrieves a paginated list of all skill aliases, ordered by name.
SELECT * FROM skill_aliases
ORDER BY alias_name
LIMIT $1
OFFSET $2;

-- name: ListAliasesForSkill :many
-- Retrieves all aliases associated with a specific canonical skill ID.
SELECT * FROM skill_aliases
WHERE skill_id = $1
ORDER BY alias_name;

-- name: UpdateSkillAlias :one
-- Updates the canonical skill a specific alias points to.
-- It's uncommon to update an alias; re-mapping is the primary use case.
UPDATE skill_aliases
SET
    skill_id = $1
WHERE alias_name = $2
RETURNING *;

-- name: DeleteSkillAlias :exec
-- Deletes a skill alias from the database by its name.
DELETE FROM skill_aliases
WHERE alias_name = $1;
