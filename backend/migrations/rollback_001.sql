-- ============================================================
-- CBT Enterprise — Rollback 001: Drop Entire Schema
-- ⚠️  DESTRUCTIVE — menghapus SEMUA data dan tabel
-- ============================================================
BEGIN;

-- Views
DROP VIEW IF EXISTS v_violation_summary CASCADE;
DROP VIEW IF EXISTS v_ujian_stats        CASCADE;
DROP VIEW IF EXISTS v_attempt_summary    CASCADE;

-- Tables (FK order: children first)
DROP TABLE IF EXISTS essay_grade       CASCADE;
DROP TABLE IF EXISTS penilaian         CASCADE;
DROP TABLE IF EXISTS proctoring_logs   CASCADE;
DROP TABLE IF EXISTS device_logs       CASCADE;
DROP TABLE IF EXISTS activity_logs     CASCADE;
DROP TABLE IF EXISTS jawaban           CASCADE;
DROP TABLE IF EXISTS attempt_soal      CASCADE;
DROP TABLE IF EXISTS attempt           CASCADE;
DROP TABLE IF EXISTS ujian_peserta     CASCADE;
DROP TABLE IF EXISTS ujian_soal        CASCADE;
DROP TABLE IF EXISTS ujian             CASCADE;
DROP TABLE IF EXISTS soal_opsi         CASCADE;
DROP TABLE IF EXISTS soal              CASCADE;
DROP TABLE IF EXISTS mapel             CASCADE;
DROP TABLE IF EXISTS kategori          CASCADE;
DROP TABLE IF EXISTS peserta           CASCADE;
DROP TABLE IF EXISTS guru              CASCADE;
DROP TABLE IF EXISTS users             CASCADE;

-- Enum types
DROP TYPE IF EXISTS scoring_status  CASCADE;
DROP TYPE IF EXISTS attempt_status  CASCADE;
DROP TYPE IF EXISTS ujian_status    CASCADE;
DROP TYPE IF EXISTS soal_tipe       CASCADE;
DROP TYPE IF EXISTS user_role       CASCADE;

-- Trigger function
DROP FUNCTION IF EXISTS set_updated_at CASCADE;

-- Extensions (hati-hati: ini bisa mempengaruhi database lain)
-- DROP EXTENSION IF EXISTS pg_trgm;
-- DROP EXTENSION IF EXISTS pgcrypto;

-- Drop migration tracker last
DROP TABLE IF EXISTS schema_migrations CASCADE;

COMMIT;
