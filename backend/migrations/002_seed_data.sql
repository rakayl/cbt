-- ============================================================
-- CBT Enterprise — Migration 002: Seed Data
-- Default mata pelajaran, kategori, dan akun admin pertama
-- ============================================================

BEGIN;

-- ── Mata Pelajaran ────────────────────────────────────────────────────────────
INSERT INTO mapel (id, nama) VALUES
    (gen_random_uuid(), 'Matematika'),
    (gen_random_uuid(), 'Fisika'),
    (gen_random_uuid(), 'Kimia'),
    (gen_random_uuid(), 'Biologi'),
    (gen_random_uuid(), 'Bahasa Indonesia'),
    (gen_random_uuid(), 'Bahasa Inggris'),
    (gen_random_uuid(), 'Sejarah'),
    (gen_random_uuid(), 'Geografi'),
    (gen_random_uuid(), 'Ekonomi'),
    (gen_random_uuid(), 'Sosiologi'),
    (gen_random_uuid(), 'Pendidikan Kewarganegaraan'),
    (gen_random_uuid(), 'Teknologi Informasi')
ON CONFLICT (nama) DO NOTHING;

-- ── Kategori Soal ─────────────────────────────────────────────────────────────
INSERT INTO kategori (id, nama) VALUES
    (gen_random_uuid(), 'Konsep Dasar'),
    (gen_random_uuid(), 'Penerapan'),
    (gen_random_uuid(), 'Analisis'),
    (gen_random_uuid(), 'Evaluasi'),
    (gen_random_uuid(), 'Penalaran'),
    (gen_random_uuid(), 'Hafalan'),
    (gen_random_uuid(), 'Perhitungan'),
    (gen_random_uuid(), 'Pemahaman Teks')
ON CONFLICT (nama) DO NOTHING;

-- ── Default Admin Account ─────────────────────────────────────────────────────
-- Password: admin123  (bcrypt cost=10)
-- GANTI password ini setelah login pertama!
INSERT INTO users (id, nama, email, password, role)
VALUES (
    gen_random_uuid(),
    'Super Admin',
    'admin@cbt.id',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    'admin'
)
ON CONFLICT (email) DO NOTHING;

-- ── Record migration ──────────────────────────────────────────────────────────
INSERT INTO schema_migrations(version,filename)
VALUES ('002','002_seed_data.sql')
ON CONFLICT (version) DO NOTHING;

COMMIT;
