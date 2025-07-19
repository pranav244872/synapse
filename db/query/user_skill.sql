-- SQLC-formatted queries for the "user_skills" junction table.
-- These follow the conventions for use with the sqlc tool.

-- name: AddSkillToUser :one
-- Adds a skill to a user with a specified proficiency level.
INSERT INTO user_skills (
    user_id,
    skill_id,
    proficiency
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetSkillsForUser :many
-- Retrieves all skills and proficiency levels for a specific user.
-- This joins with the skills table to get the skill details.
SELECT s.id, s.skill_name, us.proficiency FROM skills s
JOIN user_skills us ON s.id = us.skill_id
WHERE us.user_id = $1;

-- name: GetUsersWithSkill :many
-- Retrieves all users who have a specific skill, along with their proficiency.
-- This joins with the users table to get user details.
SELECT u.id, u.name, u.email, us.proficiency FROM users u
JOIN user_skills us ON u.id = us.user_id
WHERE us.skill_id = $1;

-- name: UpdateUserSkillProficiency :one
-- Updates a user's proficiency level for a specific skill.
UPDATE user_skills
SET proficiency = $3
WHERE user_id = $1 AND skill_id = $2
RETURNING *;

-- name: RemoveSkillFromUser :exec
-- Removes a skill from a user.
DELETE FROM user_skills
WHERE user_id = $1 AND skill_id = $2;
