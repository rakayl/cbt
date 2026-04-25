-- ============================================================
-- CBT Enterprise — Migration 001: Initial Schema
-- Idempotent (safe to re-run): uses IF NOT EXISTS / OR REPLACE
-- ============================================================

BEGIN;

-- ── Extensions ───────────────────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ── Enum types ────────────────────────────────────────────────────────────────
DO $$ BEGIN CREATE TYPE user_role AS ENUM ('admin','guru','peserta');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN CREATE TYPE soal_tipe AS ENUM ('pilihan_ganda','essay');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN CREATE TYPE ujian_status AS ENUM ('draft','aktif','selesai');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN CREATE TYPE attempt_status AS ENUM ('ongoing','paused','selesai');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN CREATE TYPE scoring_status AS ENUM ('pending','pending_essay','selesai');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ── Auto-update trigger ───────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$;

-- ═══════════════════════════════════════════════════════════════════════════════
-- USERS & ROLES
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS users (
    id          UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    nama        VARCHAR(255) NOT NULL,
    email       VARCHAR(255) NOT NULL,
    password    TEXT         NOT NULL,
    role        user_role    NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_users_email UNIQUE (email)
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role  ON users(role);
DROP TRIGGER IF EXISTS trg_users_upd ON users;
CREATE TRIGGER trg_users_upd BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS guru (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    nip        VARCHAR(50),
    mapel      VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_guru_user UNIQUE (user_id),
    CONSTRAINT fk_guru_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_guru_user ON guru(user_id);

-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS peserta (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    nis              VARCHAR(50),
    kelas            VARCHAR(50),
    foto_wajah       TEXT,
    embedding_wajah  BYTEA,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_peserta_user UNIQUE (user_id),
    CONSTRAINT fk_peserta_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_peserta_user  ON peserta(user_id);
CREATE INDEX IF NOT EXISTS idx_peserta_kelas ON peserta(kelas);

-- ═══════════════════════════════════════════════════════════════════════════════
-- BANK SOAL
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS kategori (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama VARCHAR(100) NOT NULL,
    CONSTRAINT uq_kategori_nama UNIQUE (nama)
);

CREATE TABLE IF NOT EXISTS mapel (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama    VARCHAR(100) NOT NULL,
    guru_id UUID,
    CONSTRAINT uq_mapel_nama UNIQUE (nama),
    CONSTRAINT fk_mapel_guru FOREIGN KEY (guru_id) REFERENCES guru(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_mapel_guru ON mapel(guru_id);

CREATE TABLE IF NOT EXISTS soal (
    id          UUID       PRIMARY KEY DEFAULT gen_random_uuid(),
    pertanyaan  TEXT       NOT NULL,
    tipe        soal_tipe  NOT NULL,
    kategori_id UUID,
    mapel_id    UUID,
    poin        INT        NOT NULL DEFAULT 1 CHECK (poin > 0),
    guru_id     UUID,
    pembahasan  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_soal_kategori FOREIGN KEY (kategori_id) REFERENCES kategori(id) ON DELETE SET NULL,
    CONSTRAINT fk_soal_mapel    FOREIGN KEY (mapel_id)    REFERENCES mapel(id)    ON DELETE SET NULL,
    CONSTRAINT fk_soal_guru     FOREIGN KEY (guru_id)     REFERENCES guru(id)     ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_soal_guru  ON soal(guru_id);
CREATE INDEX IF NOT EXISTS idx_soal_mapel ON soal(mapel_id);
CREATE INDEX IF NOT EXISTS idx_soal_tipe  ON soal(tipe);
CREATE INDEX IF NOT EXISTS idx_soal_trgm  ON soal USING GIN (pertanyaan gin_trgm_ops);
DROP TRIGGER IF EXISTS trg_soal_upd ON soal;
CREATE TRIGGER trg_soal_upd BEFORE UPDATE ON soal
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS soal_opsi (
    id       UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    soal_id  UUID    NOT NULL,
    teks     TEXT    NOT NULL,
    is_benar BOOLEAN NOT NULL DEFAULT FALSE,
    urutan   INT     NOT NULL DEFAULT 0,
    CONSTRAINT fk_opsi_soal FOREIGN KEY (soal_id) REFERENCES soal(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_opsi_soal ON soal_opsi(soal_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- UJIAN
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS ujian (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    judul           VARCHAR(255) NOT NULL,
    deskripsi       TEXT,
    mapel_id        UUID,
    guru_id         UUID,
    durasi_menit    INT          NOT NULL CHECK (durasi_menit > 0),
    max_peserta     INT          NOT NULL DEFAULT 0,
    status          ujian_status NOT NULL DEFAULT 'draft',
    tanggal_mulai   TIMESTAMPTZ,
    tanggal_selesai TIMESTAMPTZ,
    acak_soal       BOOLEAN      NOT NULL DEFAULT TRUE,
    acak_opsi       BOOLEAN      NOT NULL DEFAULT TRUE,
    max_tab_switch  INT          NOT NULL DEFAULT 3,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_ujian_mapel FOREIGN KEY (mapel_id) REFERENCES mapel(id) ON DELETE SET NULL,
    CONSTRAINT fk_ujian_guru  FOREIGN KEY (guru_id)  REFERENCES guru(id)  ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_ujian_guru   ON ujian(guru_id);
CREATE INDEX IF NOT EXISTS idx_ujian_status ON ujian(status);
DROP TRIGGER IF EXISTS trg_ujian_upd ON ujian;
CREATE TRIGGER trg_ujian_upd BEFORE UPDATE ON ujian
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS ujian_soal (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id UUID NOT NULL,
    soal_id  UUID NOT NULL,
    urutan   INT  NOT NULL DEFAULT 0,
    CONSTRAINT uq_ujian_soal      UNIQUE (ujian_id, soal_id),
    CONSTRAINT fk_us_ujian FOREIGN KEY (ujian_id) REFERENCES ujian(id) ON DELETE CASCADE,
    CONSTRAINT fk_us_soal  FOREIGN KEY (soal_id)  REFERENCES soal(id)  ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ujian_soal ON ujian_soal(ujian_id);

CREATE TABLE IF NOT EXISTS ujian_peserta (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id   UUID NOT NULL,
    peserta_id UUID NOT NULL,
    CONSTRAINT uq_ujian_peserta    UNIQUE (ujian_id, peserta_id),
    CONSTRAINT fk_up_ujian   FOREIGN KEY (ujian_id)   REFERENCES ujian(id)   ON DELETE CASCADE,
    CONSTRAINT fk_up_peserta FOREIGN KEY (peserta_id) REFERENCES peserta(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_up_ujian   ON ujian_peserta(ujian_id);
CREATE INDEX IF NOT EXISTS idx_up_peserta ON ujian_peserta(peserta_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- ATTEMPT (CORE EXAM SESSION)
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS attempt (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id         UUID           NOT NULL,
    peserta_id       UUID           NOT NULL,
    status           attempt_status NOT NULL DEFAULT 'ongoing',
    mulai_at         TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    selesai_at       TIMESTAMPTZ,
    sisa_detik       INT            NOT NULL DEFAULT 0,
    seed             BIGINT         NOT NULL,
    cheating_score   INT            NOT NULL DEFAULT 0,
    tab_switch_count INT            NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_attempt_ujian   FOREIGN KEY (ujian_id)   REFERENCES ujian(id)   ON DELETE RESTRICT,
    CONSTRAINT fk_attempt_peserta FOREIGN KEY (peserta_id) REFERENCES peserta(id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_attempt_ujian   ON attempt(ujian_id);
CREATE INDEX IF NOT EXISTS idx_attempt_peserta ON attempt(peserta_id);
CREATE INDEX IF NOT EXISTS idx_attempt_status  ON attempt(status);
CREATE INDEX IF NOT EXISTS idx_attempt_active
    ON attempt(peserta_id, ujian_id) WHERE status IN ('ongoing','paused');

CREATE TABLE IF NOT EXISTS attempt_soal (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL,
    soal_id    UUID NOT NULL,
    urutan     INT  NOT NULL,
    opsi_order TEXT,
    CONSTRAINT fk_as_attempt FOREIGN KEY (attempt_id) REFERENCES attempt(id) ON DELETE CASCADE,
    CONSTRAINT fk_as_soal    FOREIGN KEY (soal_id)    REFERENCES soal(id)    ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_attempt_soal ON attempt_soal(attempt_id);

CREATE TABLE IF NOT EXISTS jawaban (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id   UUID        NOT NULL,
    soal_id      UUID        NOT NULL,
    opsi_id      UUID,
    teks_jawaban TEXT,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_jawaban        UNIQUE (attempt_id, soal_id),
    CONSTRAINT fk_jaw_attempt    FOREIGN KEY (attempt_id) REFERENCES attempt(id)   ON DELETE CASCADE,
    CONSTRAINT fk_jaw_soal       FOREIGN KEY (soal_id)    REFERENCES soal(id)      ON DELETE RESTRICT,
    CONSTRAINT fk_jaw_opsi       FOREIGN KEY (opsi_id)    REFERENCES soal_opsi(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_jawaban ON jawaban(attempt_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- SCORING
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS penilaian (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id  UUID           NOT NULL,
    skor        NUMERIC(10,2)  NOT NULL DEFAULT 0,
    nilai_pg    NUMERIC(10,2)  NOT NULL DEFAULT 0,
    nilai_essay NUMERIC(10,2)  NOT NULL DEFAULT 0,
    status      scoring_status NOT NULL DEFAULT 'pending',
    grade_at    TIMESTAMPTZ,
    CONSTRAINT uq_penilaian    UNIQUE (attempt_id),
    CONSTRAINT fk_penilaian    FOREIGN KEY (attempt_id) REFERENCES attempt(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_penilaian_attempt ON penilaian(attempt_id);
CREATE INDEX IF NOT EXISTS idx_penilaian_status  ON penilaian(status);

CREATE TABLE IF NOT EXISTS essay_grade (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID          NOT NULL,
    soal_id    UUID          NOT NULL,
    nilai      NUMERIC(10,2) NOT NULL DEFAULT 0,
    catatan    TEXT,
    graded_by  UUID,
    graded_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_essay_grade     UNIQUE (attempt_id, soal_id),
    CONSTRAINT fk_eg_attempt FOREIGN KEY (attempt_id) REFERENCES attempt(id) ON DELETE CASCADE,
    CONSTRAINT fk_eg_soal    FOREIGN KEY (soal_id)    REFERENCES soal(id)    ON DELETE RESTRICT,
    CONSTRAINT fk_eg_guru    FOREIGN KEY (graded_by)  REFERENCES users(id)   ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_essay_grade ON essay_grade(attempt_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- LOGS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS activity_logs (
    id         BIGSERIAL    PRIMARY KEY,
    attempt_id UUID,
    peserta_id UUID,
    event      VARCHAR(100) NOT NULL,
    detail     TEXT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_al_attempt FOREIGN KEY (attempt_id) REFERENCES attempt(id) ON DELETE SET NULL,
    CONSTRAINT fk_al_peserta FOREIGN KEY (peserta_id) REFERENCES peserta(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_al_attempt ON activity_logs(attempt_id);
CREATE INDEX IF NOT EXISTS idx_al_peserta ON activity_logs(peserta_id);
CREATE INDEX IF NOT EXISTS idx_al_event   ON activity_logs(event);
CREATE INDEX IF NOT EXISTS idx_al_created ON activity_logs(created_at DESC);

CREATE TABLE IF NOT EXISTS device_logs (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    UUID,
    ip_address VARCHAR(50),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_dl_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_dl_user    ON device_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_dl_created ON device_logs(created_at DESC);

CREATE TABLE IF NOT EXISTS proctoring_logs (
    id         BIGSERIAL    PRIMARY KEY,
    attempt_id UUID,
    event_type VARCHAR(50)  NOT NULL,
    confidence NUMERIC(5,4) DEFAULT 0,
    image_path TEXT,
    metadata   JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_pl_attempt FOREIGN KEY (attempt_id) REFERENCES attempt(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_pl_attempt ON proctoring_logs(attempt_id);
CREATE INDEX IF NOT EXISTS idx_pl_event   ON proctoring_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_pl_created ON proctoring_logs(created_at DESC);

-- ═══════════════════════════════════════════════════════════════════════════════
-- VIEWS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE VIEW v_attempt_summary AS
SELECT
    a.id             AS attempt_id,
    a.status         AS attempt_status,
    a.mulai_at, a.selesai_at, a.sisa_detik,
    a.cheating_score, a.tab_switch_count, a.seed,
    u.id             AS ujian_id,
    u.judul          AS ujian_judul,
    u.durasi_menit,
    p.id             AS peserta_id,
    us.nama          AS peserta_nama,
    p.nis, p.kelas,
    pn.skor, pn.nilai_pg, pn.nilai_essay,
    pn.status        AS scoring_status
FROM attempt a
JOIN ujian    u  ON a.ujian_id   = u.id
JOIN peserta  p  ON a.peserta_id = p.id
JOIN users    us ON p.user_id    = us.id
LEFT JOIN penilaian pn ON pn.attempt_id = a.id;

CREATE OR REPLACE VIEW v_ujian_stats AS
SELECT
    u.id AS ujian_id, u.judul, u.status,
    COUNT(DISTINCT up.peserta_id)                                 AS total_peserta,
    COUNT(DISTINCT a.id)                                          AS total_attempt,
    COUNT(DISTINCT a.id) FILTER (WHERE a.status='ongoing')        AS ongoing,
    COUNT(DISTINCT a.id) FILTER (WHERE a.status='paused')         AS paused,
    COUNT(DISTINCT a.id) FILTER (WHERE a.status='selesai')        AS selesai,
    ROUND(AVG(pn.skor),2)                                         AS avg_skor,
    MAX(pn.skor)                                                  AS max_skor,
    MIN(pn.skor)                                                  AS min_skor,
    COALESCE(SUM(a.cheating_score),0)                             AS total_violations
FROM ujian u
LEFT JOIN ujian_peserta up ON up.ujian_id   = u.id
LEFT JOIN attempt       a  ON a.ujian_id    = u.id
LEFT JOIN penilaian     pn ON pn.attempt_id = a.id
GROUP BY u.id, u.judul, u.status;

CREATE OR REPLACE VIEW v_violation_summary AS
SELECT
    a.id AS attempt_id,
    u.judul, us.nama AS peserta_nama,
    a.cheating_score, a.tab_switch_count,
    COUNT(pl.id) FILTER (WHERE pl.event_type='multiple_faces')  AS face_multi,
    COUNT(pl.id) FILTER (WHERE pl.event_type='face_mismatch')   AS face_mismatch,
    COUNT(pl.id) FILTER (WHERE pl.event_type='no_face')         AS no_face,
    COUNT(pl.id) FILTER (WHERE pl.event_type='tab_switch')      AS tab_switch,
    COUNT(pl.id) FILTER (WHERE pl.event_type='fullscreen_exit') AS fs_exit
FROM attempt a
JOIN ujian   u  ON a.ujian_id   = u.id
JOIN peserta p  ON a.peserta_id = p.id
JOIN users   us ON p.user_id    = us.id
LEFT JOIN proctoring_logs pl ON pl.attempt_id = a.id
GROUP BY a.id, u.judul, us.nama, a.cheating_score, a.tab_switch_count;

-- ── Record migration ──────────────────────────────────────────────────────────
INSERT INTO schema_migrations(version,filename)
VALUES ('001','001_init.sql')
ON CONFLICT (version) DO NOTHING;

COMMIT;
