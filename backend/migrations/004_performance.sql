-- ============================================================
-- CBT Enterprise — Migration 004: Performance & Housekeeping
-- Tambahan index, constraint, dan optimasi query
-- ============================================================

BEGIN;

-- ── Composite indexes untuk query umum ───────────────────────────────────────

-- Cari attempt aktif peserta (digunakan setiap start exam)
CREATE INDEX IF NOT EXISTS idx_attempt_peserta_ujian_status
    ON attempt(peserta_id, ujian_id, status);

-- Leaderboard: nilai per ujian diurutkan
CREATE INDEX IF NOT EXISTS idx_penilaian_skor
    ON penilaian(attempt_id, skor DESC);

-- Proctoring log filter by type + time (admin dashboard)
CREATE INDEX IF NOT EXISTS idx_pl_event_created
    ON proctoring_logs(event_type, created_at DESC);

-- Activity log time-series (live monitoring)
CREATE INDEX IF NOT EXISTS idx_al_attempt_created
    ON activity_logs(attempt_id, created_at DESC);

-- Soal search by guru + mapel (bank soal page)
CREATE INDEX IF NOT EXISTS idx_soal_guru_mapel
    ON soal(guru_id, mapel_id);

-- Ujian search: guru + status (teacher dashboard list)
CREATE INDEX IF NOT EXISTS idx_ujian_guru_status
    ON ujian(guru_id, status);

-- Jawaban autosave lookup
CREATE INDEX IF NOT EXISTS idx_jawaban_attempt_soal
    ON jawaban(attempt_id, soal_id);

-- ── Check constraints belum ada di 001 ───────────────────────────────────────

-- Pastikan selesai_at > mulai_at jika diisi
ALTER TABLE attempt DROP CONSTRAINT IF EXISTS chk_attempt_dates;
ALTER TABLE attempt ADD CONSTRAINT chk_attempt_dates
    CHECK (selesai_at IS NULL OR selesai_at >= mulai_at);

-- cheating_score tidak boleh negatif
ALTER TABLE attempt DROP CONSTRAINT IF EXISTS chk_cheating_score;
ALTER TABLE attempt ADD CONSTRAINT chk_cheating_score
    CHECK (cheating_score >= 0);

-- skor penilaian tidak boleh negatif
ALTER TABLE penilaian DROP CONSTRAINT IF EXISTS chk_skor;
ALTER TABLE penilaian ADD CONSTRAINT chk_skor
    CHECK (skor >= 0 AND nilai_pg >= 0 AND nilai_essay >= 0);

-- essay_grade nilai tidak boleh negatif
ALTER TABLE essay_grade DROP CONSTRAINT IF EXISTS chk_essay_nilai;
ALTER TABLE essay_grade ADD CONSTRAINT chk_essay_nilai
    CHECK (nilai >= 0);

-- ── Nilai default kolom yang mungkin terlewat ─────────────────────────────────
ALTER TABLE attempt ALTER COLUMN cheating_score   SET DEFAULT 0;
ALTER TABLE attempt ALTER COLUMN tab_switch_count SET DEFAULT 0;
ALTER TABLE attempt ALTER COLUMN sisa_detik       SET DEFAULT 0;

-- ── Kolom tambahan yang dibutuhkan frontend ───────────────────────────────────
-- ip_address saat peserta login ke exam (device tracking)
ALTER TABLE attempt ADD COLUMN IF NOT EXISTS ip_address  VARCHAR(50);
ALTER TABLE attempt ADD COLUMN IF NOT EXISTS user_agent  TEXT;

-- Kolom is_active untuk soft-delete soal
ALTER TABLE soal ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX IF NOT EXISTS idx_soal_active ON soal(is_active) WHERE is_active = TRUE;

-- Kolom token_reset untuk fitur lupa password (opsional)
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token       VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token_exp   TIMESTAMPTZ;

-- ── Record migration ──────────────────────────────────────────────────────────
INSERT INTO schema_migrations(version,filename)
VALUES ('004','004_performance.sql')
ON CONFLICT (version) DO NOTHING;

COMMIT;
