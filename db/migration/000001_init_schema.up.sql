-- =============================================
-- Create Script (Up)
-- =============================================
-- This script creates the entire database schema from scratch,
-- including custom types, tables, comments, and foreign key constraints.

-- Section 1: Custom Data Types
-- -------------------------------------------
-- Define custom ENUM types to enforce data consistency for specific fields.
-- This ensures that only predefined values can be used.

CREATE TYPE "availability_status" AS ENUM (
  'available',
  'busy'
);

CREATE TYPE "task_status" AS ENUM (
  'open',
  'in_progress',
  'done'
);

CREATE TYPE "task_priority" AS ENUM (
  'low',
  'medium',
  'high',
  'critical'
);

CREATE TYPE "proficiency_level" AS ENUM (
  'beginner',
  'intermediate',
  'expert'
);


-- Section 2: Table Definitions
-- -------------------------------------------
-- Define all tables for the application.
-- Note: Foreign key columns are defined as 'bigint' to reference 'bigserial' primary keys.
-- Using 'bigserial' for a foreign key is incorrect as it would create a new sequence.

CREATE TABLE "teams" (
  "id" bigserial PRIMARY KEY,
  "team_name" varchar(255) UNIQUE NOT NULL
);

CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "name" varchar(255),
  "email" varchar(255) UNIQUE NOT NULL,
  "team_id" bigint,
  "availability" availability_status NOT NULL DEFAULT 'available'
);

CREATE TABLE "skills" (
  "id" bigserial PRIMARY KEY,
  "skill_name" varchar(100) UNIQUE NOT NULL,
  "is_verified" boolean NOT NULL DEFAULT false
);

CREATE TABLE "projects" (
  "id" bigserial PRIMARY KEY,
  "project_name" varchar(255) NOT NULL
);

CREATE TABLE "tasks" (
  "id" bigserial PRIMARY KEY,
  "project_id" bigint,
  "title" varchar(255) NOT NULL,
  "description" text,
  "status" task_status NOT NULL DEFAULT 'open',
  "priority" task_priority NOT NULL DEFAULT 'medium',
  "assignee_id" bigint,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "completed_at" timestamp
);

-- Junction table for the many-to-many relationship between users and skills
CREATE TABLE "user_skills" (
  "user_id" bigint,
  "skill_id" bigint,
  "proficiency" proficiency_level NOT NULL,
  PRIMARY KEY ("user_id", "skill_id") -- A composite primary key is more appropriate here
);

-- Junction table for the many-to-many relationship between tasks and skills
CREATE TABLE "task_required_skills" (
  "task_id" bigint,
  "skill_id" bigint,
  PRIMARY KEY ("task_id", "skill_id") -- A composite primary key is more appropriate here
);


-- Section 4: Table Comments
-- -------------------------------------------
-- Add comments to tables to explain their purpose in the schema.

COMMENT ON TABLE "users" IS 'The central entity representing talent. Availability is essential for task assignment.';
COMMENT ON TABLE "teams" IS 'Teams provide organizational context and allow filtering of users.';
COMMENT ON TABLE "skills" IS 'Controlled vocabulary to ensure consistency across the system.';
COMMENT ON TABLE "projects" IS 'Provides context and grouping for related tasks.';
COMMENT ON TABLE "tasks" IS 'Core transactional unit. Used by ML engine to recommend assignments.';
COMMENT ON TABLE "user_skills" IS 'Defines each user''s skill level for matching with task requirements.';
COMMENT ON TABLE "task_required_skills" IS 'Populated by NLP. Defines what skills are needed for each task.';


-- Section 5: Foreign Key Constraints
-- -------------------------------------------
-- Add foreign key constraints to enforce referential integrity between tables.
-- This ensures that relationships are valid.

ALTER TABLE "users" ADD FOREIGN KEY ("team_id") REFERENCES "teams" ("id") ON DELETE SET NULL;
ALTER TABLE "tasks" ADD FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON DELETE CASCADE;
ALTER TABLE "tasks" ADD FOREIGN KEY ("assignee_id") REFERENCES "users" ("id") ON DELETE SET NULL;
ALTER TABLE "user_skills" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE;
ALTER TABLE "user_skills" ADD FOREIGN KEY ("skill_id") REFERENCES "skills" ("id") ON DELETE CASCADE;
ALTER TABLE "task_required_skills" ADD FOREIGN KEY ("task_id") REFERENCES "tasks" ("id") ON DELETE CASCADE;
ALTER TABLE "task_required_skills" ADD FOREIGN KEY ("skill_id") REFERENCES "skills" ("id") ON DELETE CASCADE;
