-- ============================================================
-- CBT Enterprise — Rollback 002: Remove Seed Data
-- ============================================================
BEGIN;

DELETE FROM mapel    WHERE guru_id IS NULL
  AND nama IN ('Matematika','Fisika','Kimia','Biologi','Bahasa Indonesia',
               'Bahasa Inggris','Sejarah','Geografi','Ekonomi','Sosiologi',
               'Pendidikan Kewarganegaraan','Teknologi Informasi');

DELETE FROM kategori WHERE nama IN (
    'Konsep Dasar','Penerapan','Analisis','Evaluasi',
    'Penalaran','Hafalan','Perhitungan','Pemahaman Teks'
);

DELETE FROM users WHERE email = 'admin@cbt.id';

DELETE FROM schema_migrations WHERE version = '002';

COMMIT;
