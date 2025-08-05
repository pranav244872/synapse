-- SQLC-formatted queries for the "users" table.
-- These follow the conventions for use with the sqlc tool.

-- name: CreateUser :one
-- Inserts a new user into the users table.
INSERT INTO users (
    name,
    email,
    team_id,
	password_hash,
	role
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUser :one
-- Retrieves a single user by their unique ID.
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
-- Retrieves a single user by their unique email address.
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: ListUsers :many
-- Retrieves a paginated list of all users, ordered by ID.
SELECT * FROM users
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: ListUsersByTeam :many
-- Retrieves a paginated list of all users belonging to a specific team.
SELECT * FROM users
WHERE team_id = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateUser :one
-- Updates the details of a specific user.
-- Uses sqlc.narg() to allow for partial updates. Only non-NULL arguments will be used to update fields.
UPDATE users
SET
    name = coalesce(sqlc.narg(name), name),
    team_id = coalesce(sqlc.narg(team_id), team_id),
    availability = coalesce(sqlc.narg(availability), availability),
	role = coalesce(sqlc.narg(role), role)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RemoveUserFromTeam :one
UPDATE users
SET team_id = NULL
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
-- Deletes a user from the database by their ID.
DELETE FROM users
WHERE id = $1;

-- Counts the total number of users in the users table
-- name: CountUsers :one
SELECT count(*) FROM users;

-- Counts users whose name or email matches a search string and optionally filters by role
-- name: CountSearchUsers :one
SELECT count(*) FROM users 
WHERE (
    $1::text = '' OR 
    LOWER(name) LIKE LOWER($1) OR 
    LOWER(email) LIKE LOWER($1)
)
AND (
    $2::text = '' OR 
    role = $2::user_role
);

-- Retrieves a paginated list of users with team names, filtered by search string and optional role
-- name: SearchUsers :many
SELECT u.id, u.name, u.email, u.role, u.team_id, u.availability,
       t.team_name
FROM users u
LEFT JOIN teams t ON u.team_id = t.id
WHERE (
    $1::text = '' OR 
    LOWER(u.name) LIKE LOWER($1) OR 
    LOWER(u.email) LIKE LOWER($1)
)
AND (
    $2::text = '' OR 
    u.role = $2::user_role
)
ORDER BY u.id
LIMIT $3 OFFSET $4;

-- Gets a specific user's details along with their team name
-- name: GetUserWithTeamAndSkills :one
SELECT u.id, u.name, u.email, u.role, u.team_id, u.availability,
       t.team_name
FROM users u
LEFT JOIN teams t ON u.team_id = t.id
WHERE u.id = $1;

-- Retrieves all skills and proficiency levels for a specific user, ordered by skill name
-- name: GetUserSkillsForAdmin :many
SELECT s.id, s.skill_name, us.proficiency
FROM user_skills us
JOIN skills s ON us.skill_id = s.id
WHERE us.user_id = $1
ORDER BY s.skill_name;

-- Updates the role of a user and returns their updated information
-- name: UpdateUserRole :one
UPDATE users
SET role = $2
WHERE id = $1
RETURNING id, name, email, team_id, availability, password_hash, role;

-- Updates the team assignment of a user and returns their updated information
-- name: UpdateUserTeam :one
UPDATE users
SET team_id = $2
WHERE id = $1
RETURNING id, name, email, team_id, availability, password_hash, role;

-- List all engineers in a specific team, ordered by name
-- name: ListEngineersByTeam :many
SELECT id, name, email, availability 
FROM users 
WHERE team_id = $1 AND role = 'engineer'
ORDER BY name;

-- Count the number of users in a team with a specific role
-- name: CountUsersByTeamAndRole :one
SELECT count(*) FROM users 
WHERE team_id = $1 AND role = $2;

-- Count the number of engineers in a team with a specific availability status
-- name: CountUsersByTeamAndAvailability :one
SELECT count(*) FROM users 
WHERE team_id = $1 AND role = 'engineer' AND availability = $2;

-- Count the number of open, non-archived tasks for a specific team
-- name: CountOpenTasksByTeam :one
SELECT count(*) FROM tasks t
JOIN projects p ON t.project_id = p.id
WHERE p.team_id = $1 AND t.status = 'open' AND t.archived = false;

