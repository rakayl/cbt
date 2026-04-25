package domain

import (
	"time"

	"github.com/google/uuid"
)

// ─── USER & ROLES ────────────────────────────────────────────────────────────

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleGuru    Role = "guru"
	RolePeserta Role = "peserta"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Nama      string    `gorm:"not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
	Role      Role      `gorm:"type:varchar(20);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Guru struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User   User
	NIP    string
	Mapel  string
}

type Peserta struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User        User
	NIS         string
	Kelas       string
	FotoWajah   string // path to baseline face image
	EmbeddingWajah []byte `gorm:"type:bytea"` // baseline face embedding (JSON float64 array)
}

// ─── BANK SOAL ───────────────────────────────────────────────────────────────

type Kategori struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Nama string    `gorm:"not null"`
}

type Mapel struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Nama    string    `gorm:"not null"`
	GuruID  uuid.UUID `gorm:"type:uuid"`
	Guru    Guru
}

type TipeSoal string

const (
	SoalPG    TipeSoal = "pilihan_ganda"
	SoalEssay TipeSoal = "essay"
)

type Soal struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Pertanyaan string    `gorm:"type:text;not null"`
	Tipe       TipeSoal  `gorm:"type:varchar(20);not null"`
	KategoriID uuid.UUID `gorm:"type:uuid"`
	Kategori   Kategori
	MapelID    uuid.UUID `gorm:"type:uuid"`
	Mapel      Mapel
	Poin       int       `gorm:"default:1"`
	GuruID     uuid.UUID `gorm:"type:uuid"`
	Opsi       []SoalOpsi
	CreatedAt  time.Time
}

type SoalOpsi struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SoalID  uuid.UUID `gorm:"type:uuid;not null"`
	Teks    string    `gorm:"type:text;not null"`
	IsBenar bool      `gorm:"default:false"`
	Urutan  int
}

// ─── UJIAN ───────────────────────────────────────────────────────────────────

type StatusUjian string

const (
	UjianDraft     StatusUjian = "draft"
	UjianAktif     StatusUjian = "aktif"
	UjianSelesai   StatusUjian = "selesai"
)

type Ujian struct {
	ID              uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Judul           string      `gorm:"not null"`
	Deskripsi       string      `gorm:"type:text"`
	MapelID         uuid.UUID   `gorm:"type:uuid"`
	Mapel           Mapel
	GuruID          uuid.UUID   `gorm:"type:uuid"`
	DurasiMenit     int         `gorm:"not null"`
	MaxPeserta      int
	Status          StatusUjian `gorm:"type:varchar(20);default:'draft'"`
	TanggalMulai    *time.Time
	TanggalSelesai  *time.Time
	AcakSoal        bool        `gorm:"default:true"`
	AcakOpsi        bool        `gorm:"default:true"`
	MaxTabSwitch    int         `gorm:"default:3"`
	CreatedAt       time.Time
}

type UjianSoal struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UjianID uuid.UUID `gorm:"type:uuid;not null;index"`
	SoalID  uuid.UUID `gorm:"type:uuid;not null"`
	Soal    Soal
	Urutan  int
}

type UjianPeserta struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UjianID   uuid.UUID `gorm:"type:uuid;not null;index"`
	PesertaID uuid.UUID `gorm:"type:uuid;not null;index"`
	Peserta   Peserta
}

// ─── ATTEMPT ─────────────────────────────────────────────────────────────────

type StatusAttempt string

const (
	AttemptOngoing  StatusAttempt = "ongoing"
	AttemptPaused   StatusAttempt = "paused"
	AttemptSelesai  StatusAttempt = "selesai"
)

type Attempt struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UjianID        uuid.UUID     `gorm:"type:uuid;not null;index"`
	Ujian          Ujian
	PesertaID      uuid.UUID     `gorm:"type:uuid;not null;index"`
	Peserta        Peserta
	Status         StatusAttempt `gorm:"type:varchar(20);default:'ongoing'"`
	MulaiAt        time.Time
	SelesaiAt      *time.Time
	SisaDetik      int
	Seed           int64         // deterministic shuffle seed
	CheatingScore  int           `gorm:"default:0"`
	TabSwitchCount int           `gorm:"default:0"`
	CreatedAt      time.Time
}

type AttemptSoal struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AttemptID uuid.UUID `gorm:"type:uuid;not null;index"`
	SoalID    uuid.UUID `gorm:"type:uuid;not null"`
	Soal      Soal
	Urutan    int
	OpsiOrder string `gorm:"type:text"` // JSON array of opsi IDs (shuffled)
}

type Jawaban struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AttemptID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	SoalID      uuid.UUID  `gorm:"type:uuid;not null"`
	OpsiID      *uuid.UUID `gorm:"type:uuid"` // for PG
	TeksJawaban string     `gorm:"type:text"` // for essay
	UpdatedAt   time.Time
}

// ─── SCORING ─────────────────────────────────────────────────────────────────

type Penilaian struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AttemptID uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"`
	Attempt   Attempt
	Skor      float64
	NilaiPG   float64
	NilaiEssay float64
	Status    string     `gorm:"default:'pending'"` // pending, selesai
	GradeAt   *time.Time
}

// ─── LOGS ────────────────────────────────────────────────────────────────────

type ActivityLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	AttemptID uuid.UUID `gorm:"type:uuid;index"`
	PesertaID uuid.UUID `gorm:"type:uuid;index"`
	Event     string    `gorm:"not null"`
	Detail    string    `gorm:"type:text"`
	CreatedAt time.Time
}

type DeviceLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	UserID    uuid.UUID `gorm:"type:uuid;index"`
	IPAddress string
	UserAgent string
	CreatedAt time.Time
}

type ProctoringLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	AttemptID uuid.UUID `gorm:"type:uuid;index"`
	EventType string    // face_detected, multiple_faces, no_face, face_mismatch
	Confidence float64
	ImagePath  string
	CreatedAt time.Time
}
