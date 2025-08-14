-- =============================================
-- Migration Down: 0002_add_alias_table.down.sql
-- =============================================
-- This script reverts the addition of the skill aliases feature by
-- removing the 'skill_aliases' table and its associated objects.

-- Drop the skill_aliases table.
-- Using 'IF EXISTS' prevents an error if the script is run when the table is already gone.
-- The associated foreign key constraint is dropped automatically.
DROP TABLE IF EXISTS "skill_aliases";
