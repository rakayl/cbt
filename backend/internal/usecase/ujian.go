package usecase

import (
	"context"

	"cbt-enterprise/internal/domain"

	"github.com/google/uuid"
)

type UjianUsecase struct {
	ujianRepo domain.UjianRepository
	soalRepo  domain.SoalRepository
}

func NewUjianUsecase(ur domain.UjianRepository, sr domain.SoalRepository) *UjianUsecase {
	return &UjianUsecase{ujianRepo: ur, soalRepo: sr}
}

type CreateUjianInput struct {
	Judul        string
	Deskripsi    string
	MapelID      uuid.UUID
	GuruID       uuid.UUID
	DurasiMenit  int
	AcakSoal     bool
	AcakOpsi     bool
	MaxTabSwitch int
}

func (uc *UjianUsecase) CreateUjian(ctx context.Context, input CreateUjianInput) (*domain.Ujian, error) {
	ujian := &domain.Ujian{
		ID:           uuid.New(),
		Judul:        input.Judul,
		Deskripsi:    input.Deskripsi,
		MapelID:      input.MapelID,
		GuruID:       input.GuruID,
		DurasiMenit:  input.DurasiMenit,
		Status:       domain.UjianDraft,
		AcakSoal:     input.AcakSoal,
		AcakOpsi:     input.AcakOpsi,
		MaxTabSwitch: input.MaxTabSwitch,
	}
	return ujian, uc.ujianRepo.Create(ctx, ujian)
}

func (uc *UjianUsecase) GetUjian(ctx context.Context, id uuid.UUID) (*domain.Ujian, error) {
	return uc.ujianRepo.FindByID(ctx, id)
}

func (uc *UjianUsecase) ListByGuru(ctx context.Context, guruID uuid.UUID) ([]domain.Ujian, error) {
	return uc.ujianRepo.ListByGuru(ctx, guruID)
}

func (uc *UjianUsecase) UpdateUjian(ctx context.Context, ujian *domain.Ujian) error {
	return uc.ujianRepo.Update(ctx, ujian)
}

func (uc *UjianUsecase) SetStatus(ctx context.Context, id uuid.UUID, status domain.StatusUjian) error {
	ujian, err := uc.ujianRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	ujian.Status = status
	return uc.ujianRepo.Update(ctx, ujian)
}

func (uc *UjianUsecase) AddSoal(ctx context.Context, ujianID, soalID uuid.UUID, urutan int) error {
	return uc.ujianRepo.AddSoal(ctx, &domain.UjianSoal{
		ID:      uuid.New(),
		UjianID: ujianID,
		SoalID:  soalID,
		Urutan:  urutan,
	})
}

func (uc *UjianUsecase) AddPeserta(ctx context.Context, ujianID, pesertaID uuid.UUID) error {
	return uc.ujianRepo.AddPeserta(ctx, &domain.UjianPeserta{
		ID:        uuid.New(),
		UjianID:   ujianID,
		PesertaID: pesertaID,
	})
}

// Aliases for older handler compatibility
func (uc *UjianUsecase) Create(ctx context.Context, input CreateUjianInput) (*domain.Ujian, error) {
	return uc.CreateUjian(ctx, input)
}

func (uc *UjianUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Ujian, []domain.UjianSoal, error) {
	ujian, err := uc.ujianRepo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	soalList, err := uc.ujianRepo.GetSoalList(ctx, id)
	return ujian, soalList, err
}

func (uc *UjianUsecase) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.StatusUjian) error {
	return uc.SetStatus(ctx, id, status)
}
