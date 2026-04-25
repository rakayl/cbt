-- ============================================================
-- CBT Enterprise - Database Migration
-- Run with: psql $DATABASE_URL -f migrations/001_init.sql
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── USERS & ROLES ───────────────────────────────────────────────────────────

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama        VARCHAR(255) NOT NULL,
    email       VARCHAR(255) NOT NULL UNIQUE,
    password    TEXT NOT NULL,
    role        VARCHAR(20) NOT NULL CHECK (role IN ('admin','guru','peserta')),
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE guru (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    nip         VARCHAR(50),
    mapel       VARCHAR(100),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE peserta (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    nis              VARCHAR(50),
    kelas            VARCHAR(50),
    foto_wajah       TEXT,
    embedding_wajah  BYTEA,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- ─── BANK SOAL ───────────────────────────────────────────────────────────────

CREATE TABLE kategori (
    id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama  VARCHAR(100) NOT NULL
);

CREATE TABLE mapel (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama     VARCHAR(100) NOT NULL,
    guru_id  UUID REFERENCES guru(id)
);

CREATE TABLE soal (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pertanyaan   TEXT NOT NULL,
    tipe         VARCHAR(20) NOT NULL CHECK (tipe IN ('pilihan_ganda','essay')),
    kategori_id  UUID REFERENCES kategori(id),
    mapel_id     UUID REFERENCES mapel(id),
    poin         INT NOT NULL DEFAULT 1,
    guru_id      UUID REFERENCES guru(id),
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE soal_opsi (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    soal_id   UUID NOT NULL REFERENCES soal(id) ON DELETE CASCADE,
    teks      TEXT NOT NULL,
    is_benar  BOOLEAN DEFAULT FALSE,
    urutan    INT DEFAULT 0
);

-- ─── UJIAN ───────────────────────────────────────────────────────────────────

CREATE TABLE ujian (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    judul            VARCHAR(255) NOT NULL,
    deskripsi        TEXT,
    mapel_id         UUID REFERENCES mapel(id),
    guru_id          UUID REFERENCES guru(id),
    durasi_menit     INT NOT NULL,
    max_peserta      INT DEFAULT 0,
    status           VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft','aktif','selesai')),
    tanggal_mulai    TIMESTAMPTZ,
    tanggal_selesai  TIMESTAMPTZ,
    acak_soal        BOOLEAN DEFAULT TRUE,
    acak_opsi        BOOLEAN DEFAULT TRUE,
    max_tab_switch   INT DEFAULT 3,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ujian_soal (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id  UUID NOT NULL REFERENCES ujian(id) ON DELETE CASCADE,
    soal_id   UUID NOT NULL REFERENCES soal(id),
    urutan    INT DEFAULT 0,
    UNIQUE(ujian_id, soal_id)
);

CREATE TABLE ujian_peserta (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id    UUID NOT NULL REFERENCES ujian(id) ON DELETE CASCADE,
    peserta_id  UUID NOT NULL REFERENCES peserta(id),
    UNIQUE(ujian_id, peserta_id)
);

-- ─── ATTEMPT ─────────────────────────────────────────────────────────────────

CREATE TABLE attempt (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ujian_id         UUID NOT NULL REFERENCES ujian(id),
    peserta_id       UUID NOT NULL REFERENCES peserta(id),
    status           VARCHAR(20) DEFAULT 'ongoing' CHECK (status IN ('ongoing','paused','selesai')),
    mulai_at         TIMESTAMPTZ DEFAULT NOW(),
    selesai_at       TIMESTAMPTZ,
    sisa_detik       INT NOT NULL,
    seed             BIGINT NOT NULL,
    cheating_score   INT DEFAULT 0,
    tab_switch_count INT DEFAULT 0,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE attempt_soal (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id  UUID NOT NULL REFERENCES attempt(id) ON DELETE CASCADE,
    soal_id     UUID NOT NULL REFERENCES soal(id),
    urutan      INT NOT NULL,
    opsi_order  TEXT  -- JSON array of opsi IDs
);

CREATE TABLE jawaban (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id    UUID NOT NULL REFERENCES attempt(id) ON DELETE CASCADE,
    soal_id       UUID NOT NULL REFERENCES soal(id),
    opsi_id       UUID REFERENCES soal_opsi(id),
    teks_jawaban  TEXT,
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(attempt_id, soal_id)
);

-- ─── SCORING ─────────────────────────────────────────────────────────────────

CREATE TABLE penilaian (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id   UUID NOT NULL UNIQUE REFERENCES attempt(id),
    skor         NUMERIC(10,2) DEFAULT 0,
    nilai_pg     NUMERIC(10,2) DEFAULT 0,
    nilai_essay  NUMERIC(10,2) DEFAULT 0,
    status       VARCHAR(30) DEFAULT 'pending',
    grade_at     TIMESTAMPTZ
);

-- ─── LOGS ────────────────────────────────────────────────────────────────────

CREATE TABLE activity_logs (
    id          BIGSERIAL PRIMARY KEY,
    attempt_id  UUID REFERENCES attempt(id),
    peserta_id  UUID REFERENCES peserta(id),
    event       VARCHAR(100) NOT NULL,
    detail      TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE device_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users(id),
    ip_address  VARCHAR(50),
    user_agent  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE proctoring_logs (
    id          BIGSERIAL PRIMARY KEY,
    attempt_id  UUID REFERENCES attempt(id),
    event_type  VARCHAR(50),  -- face_detected, multiple_faces, no_face, face_mismatch
    confidence  NUMERIC(5,4),
    image_path  TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- ─── INDEXES ─────────────────────────────────────────────────────────────────

CREATE INDEX idx_attempt_peserta     ON attempt(peserta_id);
CREATE INDEX idx_attempt_ujian       ON attempt(ujian_id);
CREATE INDEX idx_attempt_status      ON attempt(status);
CREATE INDEX idx_jawaban_attempt     ON jawaban(attempt_id);
CREATE INDEX idx_attempt_soal_atp    ON attempt_soal(attempt_id);
CREATE INDEX idx_activity_attempt    ON activity_logs(attempt_id);
CREATE INDEX idx_proctoring_attempt  ON proctoring_logs(attempt_id);
CREATE INDEX idx_soal_mapel          ON soal(mapel_id);
CREATE INDEX idx_ujian_status        ON ujian(status);
