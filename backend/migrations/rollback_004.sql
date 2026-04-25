-- ============================================================
-- CBT Enterprise — Rollback 004: Performance & Housekeeping
-- ============================================================
BEGIN;

DROP INDEX IF EXISTS idx_attempt_peserta_ujian_status;
DROP INDEX IF EXISTS idx_penilaian_skor;
DROP INDEX IF EXISTS idx_pl_event_created;
DROP INDEX IF EXISTS idx_al_attempt_created;
DROP INDEX IF EXISTS idx_soal_guru_mapel;
DROP INDEX IF EXISTS idx_ujian_guru_status;
DROP INDEX IF EXISTS idx_jawaban_attempt_soal;

ALTER TABLE attempt   DROP CONSTRAINT IF EXISTS chk_attempt_dates;
ALTER TABLE attempt   DROP CONSTRAINT IF EXISTS chk_cheating_score;
ALTER TABLE penilaian DROP CONSTRAINT IF EXISTS chk_skor;
ALTER TABLE essay_grade DROP CONSTRAINT IF EXISTS chk_essay_nilai;

ALTER TABLE attempt DROP COLUMN IF EXISTS ip_address;
ALTER TABLE attempt DROP COLUMN IF EXISTS user_agent;
ALTER TABLE soal    DROP COLUMN IF EXISTS is_active;
ALTER TABLE users   DROP COLUMN IF EXISTS reset_token;
ALTER TABLE users   DROP COLUMN IF EXISTS reset_token_exp;

DELETE FROM schema_migrations WHERE version = '004';

COMMIT;
