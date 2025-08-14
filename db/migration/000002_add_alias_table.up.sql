-- =============================================
-- Migration Up: 0002_add_alias_table.up.sql
-- =============================================
-- This script introduces a table for skill aliases (or synonyms).
-- This allows mapping different terms (e.g., "JS", "ECMAScript") to a canonical skill ("JavaScript"),
-- which is crucial for improving NLP processing and search functionality.

-- Section 1: Table Definition
-- -------------------------------------------
-- Create the skill_aliases table to store synonyms for skills.
-- The alias_name is the primary key to ensure uniqueness.
CREATE TABLE "skill_aliases" (
    "alias_name" varchar(100) PRIMARY KEY,
    "skill_id" bigint NOT NULL,
    CONSTRAINT "fk_skill_aliases_to_skills"
        FOREIGN KEY("skill_id") 
        REFERENCES "skills"("id") 
        ON DELETE CASCADE
);

-- Section 3: Table Comments
-- -------------------------------------------
-- Add a comment to the table to explain its purpose in the schema.
COMMENT ON TABLE "skill_aliases" IS 'Maps alternative names or synonyms to a canonical skill in the skills table. Used by LLM to normalize task requirements.';
