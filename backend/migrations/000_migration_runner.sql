-- ============================================================
-- CBT Enterprise — Migration: 000_migration_runner
-- Run FIRST before any other migration.
-- Creates the schema_migrations tracking table.
-- ============================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     VARCHAR(14)  PRIMARY KEY,          -- e.g. "001"
    filename    VARCHAR(255) NOT NULL,
    applied_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    checksum    VARCHAR(64)                          -- SHA-256 of file (optional)
);

COMMENT ON TABLE schema_migrations IS
    'Tracks applied database migrations for CBT Enterprise.';
