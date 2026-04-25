package domain

import (
	"context"

	"github.com/google/uuid"
)

// ─── AUTH ─────────────────────────────────────────────────────────────────────

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type PesertaRepository interface {
	Create(ctx context.Context, p *Peserta) error
	FindByUserID(ctx context.Context, userID uuid.UUID) (*Peserta, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Peserta, error)
	UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []byte) error
	ListAll(ctx context.Context) ([]Peserta, error)
}

type GuruRepository interface {
	Create(ctx context.Context, g *Guru) error
	FindByUserID(ctx context.Context, userID uuid.UUID) (*Guru, error)
}

// ─── BANK SOAL ───────────────────────────────────────────────────────────────

type SoalRepository interface {
	Create(ctx context.Context, soal *Soal) error
	FindByID(ctx context.Context, id uuid.UUID) (*Soal, error)
	FindByMapel(ctx context.Context, mapelID uuid.UUID) ([]Soal, error)
	Update(ctx context.Context, soal *Soal) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByGuru(ctx context.Context, guruID uuid.UUID) ([]Soal, error)
}

// ─── UJIAN ───────────────────────────────────────────────────────────────────

type UjianRepository interface {
	Create(ctx context.Context, ujian *Ujian) error
	FindByID(ctx context.Context, id uuid.UUID) (*Ujian, error)
	Update(ctx context.Context, ujian *Ujian) error
	ListAktif(ctx context.Context) ([]Ujian, error)
	ListByGuru(ctx context.Context, guruID uuid.UUID) ([]Ujian, error)
	AddSoal(ctx context.Context, us *UjianSoal) error
	GetSoalList(ctx context.Context, ujianID uuid.UUID) ([]UjianSoal, error)
	AddPeserta(ctx context.Context, up *UjianPeserta) error
	GetPesertaList(ctx context.Context, ujianID uuid.UUID) ([]UjianPeserta, error)
}

// ─── ATTEMPT ─────────────────────────────────────────────────────────────────

type AttemptRepository interface {
	Create(ctx context.Context, a *Attempt) error
	FindByID(ctx context.Context, id uuid.UUID) (*Attempt, error)
	FindActive(ctx context.Context, pesertaID, ujianID uuid.UUID) (*Attempt, error)
	Update(ctx context.Context, a *Attempt) error
	SaveAttemptSoal(ctx context.Context, soalList []AttemptSoal) error
	GetAttemptSoal(ctx context.Context, attemptID uuid.UUID) ([]AttemptSoal, error)
	UpsertJawaban(ctx context.Context, j *Jawaban) error
	GetJawaban(ctx context.Context, attemptID uuid.UUID) ([]Jawaban, error)
	IncrementCheating(ctx context.Context, id uuid.UUID) error
	PauseAttempt(ctx context.Context, id uuid.UUID) error
	FinishAttempt(ctx context.Context, id uuid.UUID) error
	ListByUjian(ctx context.Context, ujianID uuid.UUID) ([]Attempt, error)
}

// ─── SCORING ─────────────────────────────────────────────────────────────────

type PenilaianRepository interface {
	Create(ctx context.Context, p *Penilaian) error
	Update(ctx context.Context, p *Penilaian) error
	FindByAttempt(ctx context.Context, attemptID uuid.UUID) (*Penilaian, error)
}

// ─── LOGS ─────────────────────────────────────────────────────────────────────

type ActivityLogRepository interface {
	Log(ctx context.Context, log *ActivityLog) error
	FindByAttempt(ctx context.Context, attemptID uuid.UUID) ([]ActivityLog, error)
}

type ProctoringLogRepository interface {
	Log(ctx context.Context, log *ProctoringLog) error
	FindByAttempt(ctx context.Context, attemptID uuid.UUID) ([]ProctoringLog, error)
}
