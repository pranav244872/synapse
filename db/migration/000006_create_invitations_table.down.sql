-- =============================================
-- Migration Down: 000006_create_invitations_table.down.sql
-- =============================================
-- This migration reverts the creation of the 'invitations' table, completely
-- removing it and its associated data and constraints from the database.

-- Section 1: Drop Invitations Table
-- -------------------------------------------
-- This single command drops the table, along with its primary key,
-- indexes, constraints, and defaults. Using IF EXISTS prevents an error
-- if the table has already been removed.
DROP TABLE IF EXISTS invitations;
