-- =============================================
-- Drop Script (Down)
-- =============================================
-- This script will drop all tables, and custom types defined in the schema.
-- It is written to be run safely and handle dependencies.

-- foreign key constraints, and any other dependent objects. Explicitly dropping
-- constraints first is not necessary with this approach.

-- Drop Tables
-- We use 'IF EXISTS' to prevent errors if the tables have already been removed.
-- We use 'CASCADE' to automatically remove any dependent objects, such as foreign keys.
-- The order is from most dependent to least dependent, which is a good practice,
-- although CASCADE makes the order less critical.

COMMENT ON TABLE "task_required_skills" IS NULL;
DROP TABLE IF EXISTS "task_required_skills" CASCADE;

COMMENT ON TABLE "user_skills" IS NULL;
DROP TABLE IF EXISTS "user_skills" CASCADE;

COMMENT ON TABLE "tasks" IS NULL;
DROP TABLE IF EXISTS "tasks" CASCADE;

COMMENT ON TABLE "projects" IS NULL;
DROP TABLE IF EXISTS "projects" CASCADE;

COMMENT ON TABLE "skills" IS NULL;
DROP TABLE IF EXISTS "skills" CASCADE;

COMMENT ON TABLE "users" IS NULL;
DROP TABLE IF EXISTS "users" CASCADE;

COMMENT ON TABLE "teams" IS NULL;
DROP TABLE IF EXISTS "teams" CASCADE;


-- Drop Custom ENUM Types
-- These can only be dropped after the tables that use them are gone.

DROP TYPE IF EXISTS "proficiency_level";
DROP TYPE IF EXISTS "task_priority";
DROP TYPE IF EXISTS "task_status";
DROP TYPE IF EXISTS "availability_status";
