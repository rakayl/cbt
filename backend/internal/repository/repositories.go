package repository

import (
	"context"

	"cbt-enterprise/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── User ──────────────────────────────────────────────────────────────────────

type userRepo struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) domain.UserRepository { return &userRepo{db} }

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}
func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	return &u, r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
}
func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	return &u, r.db.WithContext(ctx).First(&u, "id = ?", id).Error
}

// ── Peserta ───────────────────────────────────────────────────────────────────

type pesertaRepo struct{ db *gorm.DB }

func NewPesertaRepository(db *gorm.DB) domain.PesertaRepository { return &pesertaRepo{db} }

func (r *pesertaRepo) Create(ctx context.Context, p *domain.Peserta) error {
	return r.db.WithContext(ctx).Create(p).Error
}
func (r *pesertaRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Peserta, error) {
	var p domain.Peserta
	return &p, r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).First(&p).Error
}
func (r *pesertaRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Peserta, error) {
	var p domain.Peserta
	return &p, r.db.WithContext(ctx).Preload("User").First(&p, "id = ?", id).Error
}
func (r *pesertaRepo) UpdateEmbedding(ctx context.Context, id uuid.UUID, emb []byte) error {
	return r.db.WithContext(ctx).Model(&domain.Peserta{}).Where("id = ?", id).Update("embedding_wajah", emb).Error
}
func (r *pesertaRepo) ListAll(ctx context.Context) ([]domain.Peserta, error) {
	var list []domain.Peserta
	return list, r.db.WithContext(ctx).Preload("User").Find(&list).Error
}

// ── Guru ──────────────────────────────────────────────────────────────────────

type guruRepo struct{ db *gorm.DB }

func NewGuruRepository(db *gorm.DB) domain.GuruRepository { return &guruRepo{db} }

func (r *guruRepo) Create(ctx context.Context, g *domain.Guru) error {
	return r.db.WithContext(ctx).Create(g).Error
}
func (r *guruRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Guru, error) {
	var g domain.Guru
	return &g, r.db.WithContext(ctx).Where("user_id = ?", userID).First(&g).Error
}

// ── Soal ──────────────────────────────────────────────────────────────────────

type soalRepo struct{ db *gorm.DB }

func NewSoalRepository(db *gorm.DB) domain.SoalRepository { return &soalRepo{db} }

func (r *soalRepo) Create(ctx context.Context, s *domain.Soal) error {
	return r.db.WithContext(ctx).Create(s).Error
}
func (r *soalRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Soal, error) {
	var s domain.Soal
	return &s, r.db.WithContext(ctx).Preload("Opsi").First(&s, "id = ?", id).Error
}
func (r *soalRepo) FindByMapel(ctx context.Context, mapelID uuid.UUID) ([]domain.Soal, error) {
	var list []domain.Soal
	return list, r.db.WithContext(ctx).Preload("Opsi").Where("mapel_id = ?", mapelID).Find(&list).Error
}
func (r *soalRepo) Update(ctx context.Context, s *domain.Soal) error {
	return r.db.WithContext(ctx).Save(s).Error
}
func (r *soalRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Soal{}, "id = ?", id).Error
}
func (r *soalRepo) ListByGuru(ctx context.Context, guruID uuid.UUID) ([]domain.Soal, error) {
	var list []domain.Soal
	return list, r.db.WithContext(ctx).Preload("Opsi").Where("guru_id = ?", guruID).Find(&list).Error
}

// ── Ujian ─────────────────────────────────────────────────────────────────────

type ujianRepo struct{ db *gorm.DB }

func NewUjianRepository(db *gorm.DB) domain.UjianRepository { return &ujianRepo{db} }

func (r *ujianRepo) Create(ctx context.Context, u *domain.Ujian) error {
	return r.db.WithContext(ctx).Create(u).Error
}
func (r *ujianRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Ujian, error) {
	var u domain.Ujian
	return &u, r.db.WithContext(ctx).First(&u, "id = ?", id).Error
}
func (r *ujianRepo) Update(ctx context.Context, u *domain.Ujian) error {
	return r.db.WithContext(ctx).Save(u).Error
}
func (r *ujianRepo) ListAktif(ctx context.Context) ([]domain.Ujian, error) {
	var list []domain.Ujian
	return list, r.db.WithContext(ctx).Where("status = ?", domain.UjianAktif).Find(&list).Error
}
func (r *ujianRepo) ListByGuru(ctx context.Context, guruID uuid.UUID) ([]domain.Ujian, error) {
	var list []domain.Ujian
	return list, r.db.WithContext(ctx).Where("guru_id = ?", guruID).Find(&list).Error
}
func (r *ujianRepo) AddSoal(ctx context.Context, us *domain.UjianSoal) error {
	return r.db.WithContext(ctx).Create(us).Error
}
func (r *ujianRepo) GetSoalList(ctx context.Context, ujianID uuid.UUID) ([]domain.UjianSoal, error) {
	var list []domain.UjianSoal
	return list, r.db.WithContext(ctx).Preload("Soal.Opsi").
		Where("ujian_id = ?", ujianID).Order("urutan").Find(&list).Error
}
func (r *ujianRepo) AddPeserta(ctx context.Context, up *domain.UjianPeserta) error {
	return r.db.WithContext(ctx).Create(up).Error
}
func (r *ujianRepo) GetPesertaList(ctx context.Context, ujianID uuid.UUID) ([]domain.UjianPeserta, error) {
	var list []domain.UjianPeserta
	return list, r.db.WithContext(ctx).Preload("Peserta.User").
		Where("ujian_id = ?", ujianID).Find(&list).Error
}

// ── Attempt ───────────────────────────────────────────────────────────────────

type attemptRepo struct{ db *gorm.DB }

func NewAttemptRepository(db *gorm.DB) domain.AttemptRepository { return &attemptRepo{db} }

func (r *attemptRepo) Create(ctx context.Context, a *domain.Attempt) error {
	return r.db.WithContext(ctx).Create(a).Error
}
func (r *attemptRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attempt, error) {
	var a domain.Attempt
	return &a, r.db.WithContext(ctx).First(&a, "id = ?", id).Error
}
func (r *attemptRepo) FindActive(ctx context.Context, pesertaID, ujianID uuid.UUID) (*domain.Attempt, error) {
	var a domain.Attempt
	return &a, r.db.WithContext(ctx).
		Where("peserta_id = ? AND ujian_id = ? AND status IN ?",
			pesertaID, ujianID, []string{"ongoing", "paused"}).
		First(&a).Error
}
func (r *attemptRepo) Update(ctx context.Context, a *domain.Attempt) error {
	return r.db.WithContext(ctx).Save(a).Error
}
func (r *attemptRepo) SaveAttemptSoal(ctx context.Context, list []domain.AttemptSoal) error {
	return r.db.WithContext(ctx).Create(&list).Error
}
func (r *attemptRepo) GetAttemptSoal(ctx context.Context, attemptID uuid.UUID) ([]domain.AttemptSoal, error) {
	var list []domain.AttemptSoal
	return list, r.db.WithContext(ctx).Preload("Soal.Opsi").
		Where("attempt_id = ?", attemptID).Order("urutan").Find(&list).Error
}
func (r *attemptRepo) UpsertJawaban(ctx context.Context, j *domain.Jawaban) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "attempt_id"}, {Name: "soal_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"opsi_id", "teks_jawaban", "updated_at"}),
	}).Create(j).Error
}
func (r *attemptRepo) GetJawaban(ctx context.Context, attemptID uuid.UUID) ([]domain.Jawaban, error) {
	var list []domain.Jawaban
	return list, r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).Find(&list).Error
}
func (r *attemptRepo) IncrementCheating(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.Attempt{}).
		Where("id = ?", id).
		UpdateColumn("cheating_score", gorm.Expr("cheating_score + 1")).Error
}
func (r *attemptRepo) PauseAttempt(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.Attempt{}).
		Where("id = ?", id).Update("status", domain.AttemptPaused).Error
}
func (r *attemptRepo) FinishAttempt(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.Attempt{}).
		Where("id = ?", id).Updates(map[string]interface{}{
		"status":     domain.AttemptSelesai,
		"selesai_at": gorm.Expr("NOW()"),
	}).Error
}
func (r *attemptRepo) ListByUjian(ctx context.Context, ujianID uuid.UUID) ([]domain.Attempt, error) {
	var list []domain.Attempt
	return list, r.db.WithContext(ctx).Where("ujian_id = ?", ujianID).Find(&list).Error
}

// ── Penilaian ─────────────────────────────────────────────────────────────────

type penilaianRepo struct{ db *gorm.DB }

func NewPenilaianRepository(db *gorm.DB) domain.PenilaianRepository { return &penilaianRepo{db} }

func (r *penilaianRepo) Create(ctx context.Context, p *domain.Penilaian) error {
	return r.db.WithContext(ctx).Create(p).Error
}
func (r *penilaianRepo) Update(ctx context.Context, p *domain.Penilaian) error {
	return r.db.WithContext(ctx).Save(p).Error
}
func (r *penilaianRepo) FindByAttempt(ctx context.Context, attemptID uuid.UUID) (*domain.Penilaian, error) {
	var p domain.Penilaian
	return &p, r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).First(&p).Error
}

// ── ActivityLog ───────────────────────────────────────────────────────────────

type activityLogRepo struct{ db *gorm.DB }

func NewActivityLogRepository(db *gorm.DB) domain.ActivityLogRepository {
	return &activityLogRepo{db}
}
func (r *activityLogRepo) Log(ctx context.Context, l *domain.ActivityLog) error {
	return r.db.WithContext(ctx).Create(l).Error
}
func (r *activityLogRepo) FindByAttempt(ctx context.Context, attemptID uuid.UUID) ([]domain.ActivityLog, error) {
	var list []domain.ActivityLog
	return list, r.db.WithContext(ctx).Where("attempt_id = ?", attemptID).
		Order("created_at desc").Find(&list).Error
}
