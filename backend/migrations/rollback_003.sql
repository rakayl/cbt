-- ============================================================
-- CBT Enterprise — Rollback 003: Remove Sample Data
-- ============================================================
BEGIN;

-- Hapus dalam urutan dependency (FK dari bawah ke atas)
DELETE FROM ujian_peserta WHERE ujian_id = '66666666-0000-0000-0000-000000000001';
DELETE FROM ujian_soal    WHERE ujian_id = '66666666-0000-0000-0000-000000000001';
DELETE FROM ujian         WHERE id       = '66666666-0000-0000-0000-000000000001';

DELETE FROM soal_opsi WHERE soal_id IN (
    SELECT id FROM soal WHERE guru_id = '22222222-0000-0000-0000-000000000001'
);
DELETE FROM soal WHERE guru_id = '22222222-0000-0000-0000-000000000001';

DELETE FROM mapel   WHERE nama = 'Matematika Demo';
DELETE FROM peserta WHERE user_id IN (
    SELECT id FROM users WHERE email LIKE 'peserta%@cbt.id'
);
DELETE FROM guru    WHERE user_id = '11111111-0000-0000-0000-000000000001';
DELETE FROM users   WHERE email IN (
    'guru@cbt.id','peserta1@cbt.id','peserta2@cbt.id',
    'peserta3@cbt.id','peserta4@cbt.id','peserta5@cbt.id'
);

DELETE FROM schema_migrations WHERE version = '003';

COMMIT;
