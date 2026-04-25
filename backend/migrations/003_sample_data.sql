-- ============================================================
-- CBT Enterprise — Migration 003: Sample / Demo Data
-- Guru, Peserta, Soal, Ujian lengkap untuk testing
-- Jalankan HANYA di environment development/staging
-- ============================================================

BEGIN;

-- ── Demo Guru ─────────────────────────────────────────────────────────────────
-- Password: guru123
INSERT INTO users (id, nama, email, password, role) VALUES
    ('11111111-0000-0000-0000-000000000001', 'Ibu Sari Dewi, S.Pd', 'guru@cbt.id',
     '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'guru')
ON CONFLICT (email) DO NOTHING;

INSERT INTO guru (id, user_id, nip, mapel) VALUES
    ('22222222-0000-0000-0000-000000000001',
     '11111111-0000-0000-0000-000000000001',
     '198501012010012001', 'Matematika')
ON CONFLICT (user_id) DO NOTHING;

-- ── Demo Peserta ──────────────────────────────────────────────────────────────
-- Password: peserta123
INSERT INTO users (id, nama, email, password, role) VALUES
    ('11111111-0000-0000-0000-000000000002', 'Andi Pratama',    'peserta1@cbt.id', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'peserta'),
    ('11111111-0000-0000-0000-000000000003', 'Budi Santoso',    'peserta2@cbt.id', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'peserta'),
    ('11111111-0000-0000-0000-000000000004', 'Citra Lestari',   'peserta3@cbt.id', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'peserta'),
    ('11111111-0000-0000-0000-000000000005', 'Dewi Rahayu',     'peserta4@cbt.id', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'peserta'),
    ('11111111-0000-0000-0000-000000000006', 'Eko Purnomo',     'peserta5@cbt.id', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'peserta')
ON CONFLICT (email) DO NOTHING;

INSERT INTO peserta (id, user_id, nis, kelas) VALUES
    ('33333333-0000-0000-0000-000000000001', '11111111-0000-0000-0000-000000000002', '2024001', 'XII IPA 1'),
    ('33333333-0000-0000-0000-000000000002', '11111111-0000-0000-0000-000000000003', '2024002', 'XII IPA 1'),
    ('33333333-0000-0000-0000-000000000003', '11111111-0000-0000-0000-000000000004', '2024003', 'XII IPA 2'),
    ('33333333-0000-0000-0000-000000000004', '11111111-0000-0000-0000-000000000005', '2024004', 'XII IPS 1'),
    ('33333333-0000-0000-0000-000000000005', '11111111-0000-0000-0000-000000000006', '2024005', 'XII IPS 1')
ON CONFLICT (user_id) DO NOTHING;

-- ── Demo Mata Pelajaran (with guru link) ──────────────────────────────────────
INSERT INTO mapel (id, nama, guru_id) VALUES
    ('44444444-0000-0000-0000-000000000001', 'Matematika Demo',
     '22222222-0000-0000-0000-000000000001')
ON CONFLICT (nama) DO NOTHING;

-- ── Demo Soal Pilihan Ganda ───────────────────────────────────────────────────
DO $$
DECLARE
    s1 UUID := '55555555-0000-0000-0000-000000000001';
    s2 UUID := '55555555-0000-0000-0000-000000000002';
    s3 UUID := '55555555-0000-0000-0000-000000000003';
    s4 UUID := '55555555-0000-0000-0000-000000000004';
    s5 UUID := '55555555-0000-0000-0000-000000000005';
    s6 UUID := '55555555-0000-0000-0000-000000000006';
    s7 UUID := '55555555-0000-0000-0000-000000000007';
    mapel_id UUID := '44444444-0000-0000-0000-000000000001';
    guru_id  UUID := '22222222-0000-0000-0000-000000000001';
BEGIN
    -- Soal 1: Limit
    INSERT INTO soal(id,pertanyaan,tipe,mapel_id,poin,guru_id,pembahasan) VALUES
    (s1,'Nilai dari lim(x→2) (x² - 4)/(x - 2) adalah...','pilihan_ganda',mapel_id,2,guru_id,
     'Faktorkan: (x-2)(x+2)/(x-2) = x+2. Substitusi x=2: 4.')
    ON CONFLICT DO NOTHING;
    INSERT INTO soal_opsi(soal_id,teks,is_benar,urutan) VALUES
    (s1,'4',  TRUE,  1),(s1,'2',  FALSE, 2),(s1,'0',  FALSE, 3),(s1,'∞',  FALSE, 4)
    ON CONFLICT DO NOTHING;

    -- Soal 2: Turunan
    INSERT INTO soal(id,pertanyaan,tipe,mapel_id,poin,guru_id,pembahasan) VALUES
    (s2,'Turunan pertama dari f(x) = 3x³ - 2x² + 5x - 1 adalah...','pilihan_ganda',mapel_id,2,guru_id,
     'f''(x) = 9x² - 4x + 5')
    ON CONFLICT DO NOTHING;
    INSERT INTO soal_opsi(soal_id,teks,is_benar,urutan) VALUES
    (s2,'9x² - 4x + 5', TRUE, 1),(s2,'9x² + 4x - 5', FALSE, 2),
    (s2,'3x² - 2x + 5', FALSE, 3),(s2,'9x² - 4x - 5', FALSE, 4)
    ON CONFLICT DO NOTHING;

    -- Soal 3: Integral
    INSERT INTO soal(id,pertanyaan,tipe,mapel_id,poin,guru_id,pembahasan) VALUES
    (s3,'Hasil dari ∫₀² (3x² + 2x) dx adalah...','pilihan_ganda',mapel_id,3,guru_id,
     '[x³ + x²]₀² = (8+4) - (0) = 12')
    ON CONFLICT DO NOTHING;
    INSERT INTO soal_opsi(soal_id,teks,is_benar,urutan) VALUES
    (s3,'12', TRUE, 1),(s3,'14', FALSE, 2),(s3,'10', FALSE, 3),(s3,'16', FALSE, 4)
    ON CONFLICT DO NOTHING;

    -- Soal 4: Fungsi minimum
    INSERT INTO soal(id,pertanyaan,tipe,mapel_id,poin,guru_id,pembahasan) VALUES
    (s4,'Diketahui f(x) = 3x² - 6x + 4. Nilai minimum fungsi tersebut adalah...','pilihan_ganda',mapel_id,2,guru_id,
     'x_min = -b/2a = 6/6 = 1. f(1) = 3-6+4 = 1')
    ON CONFLICT DO NOTHING;
    INSERT INTO soal_opsi(soal_id,teks,is_benar,urutan) VALUES
    (s4,'1', TRUE, 1),(s4,'2', FALSE, 2),(s4,'3', FALSE, 3),(s4,'4', FALSE, 4)
    ON CONFLICT DO NOTHING;

    -- Soal 5: Matriks
    INSERT INTO soal(id,pertanyaan,tipe,mapel_id,poin,guru_id,pembahasan) VALUES
    (s5,'Jika matriks A = [[1,2],[3,4]], maka determinan A adalah...','pilihan_ganda',mapel_id,2,guru_id,
     'det = (1×4) - (2×3) = 4 - 6 = -2')
    ON CONFLICT DO NOTHING;
    INSERT INTO soal_opsi(soal_id,teks,is_benar,urutan) VALUES
    (s5,'-2', TRUE, 1),(s5,'2', FALSE, 2),(s5,'-10', FALSE, 3),(s5,'10', FALSE, 4)
    ON CONFLICT DO NOTHING;

    -- Soal 6: Vektor
    INSERT INTO soal(id,pertanyaan,tipe,mapel_id,poin,guru_id,pembahasan) VALUES
    (s6,'Jika vektor a = (3,4), maka |a| adalah...','pilihan_ganda',mapel_id,2,guru_id,
     '|a| = √(3²+4²) = √25 = 5')
    ON CONFLICT DO NOTHING;
    INSERT INTO soal_opsi(soal_id,teks,is_benar,urutan) VALUES
    (s6,'5', TRUE, 1),(s6,'7', FALSE, 2),(s6,'12', FALSE, 3),(s6,'25', FALSE, 4)
    ON CONFLICT DO NOTHING;

    -- Soal 7: Essay
    INSERT INTO soal(id,pertanyaan,tipe,mapel_id,poin,guru_id) VALUES
    (s7,'Jelaskan konsep limit fungsi dalam kalkulus dan berikan dua contoh penerapannya dalam kehidupan nyata!',
     'essay',mapel_id,10,guru_id)
    ON CONFLICT DO NOTHING;
END $$;

-- ── Demo Ujian ────────────────────────────────────────────────────────────────
INSERT INTO ujian (id,judul,deskripsi,mapel_id,guru_id,durasi_menit,status,acak_soal,acak_opsi,max_tab_switch)
VALUES (
    '66666666-0000-0000-0000-000000000001',
    'UTS Matematika Kelas XII',
    'Ujian tengah semester materi kalkulus, turunan, dan integral',
    '44444444-0000-0000-0000-000000000001',
    '22222222-0000-0000-0000-000000000001',
    60, 'aktif', TRUE, TRUE, 3
) ON CONFLICT DO NOTHING;

-- Tambah semua soal ke ujian
INSERT INTO ujian_soal (ujian_id, soal_id, urutan)
SELECT '66666666-0000-0000-0000-000000000001', id, ROW_NUMBER() OVER (ORDER BY created_at)
FROM soal WHERE guru_id = '22222222-0000-0000-0000-000000000001'
ON CONFLICT DO NOTHING;

-- Daftarkan semua peserta ke ujian
INSERT INTO ujian_peserta (ujian_id, peserta_id)
SELECT '66666666-0000-0000-0000-000000000001', id
FROM peserta
ON CONFLICT DO NOTHING;

-- ── Record migration ──────────────────────────────────────────────────────────
INSERT INTO schema_migrations(version,filename)
VALUES ('003','003_sample_data.sql')
ON CONFLICT (version) DO NOTHING;

COMMIT;
