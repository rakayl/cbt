package db

import (
	"log"

	"cbt-enterprise/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgres(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	return db
}

func AutoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&domain.User{},
		&domain.Guru{},
		&domain.Peserta{},
		&domain.Kategori{},
		&domain.Mapel{},
		&domain.Soal{},
		&domain.SoalOpsi{},
		&domain.Ujian{},
		&domain.UjianSoal{},
		&domain.UjianPeserta{},
		&domain.Attempt{},
		&domain.AttemptSoal{},
		&domain.Jawaban{},
		&domain.Penilaian{},
		&domain.ActivityLog{},
		&domain.DeviceLog{},
		&domain.ProctoringLog{},
	)
	if err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	// Add composite unique index on jawaban
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_jawaban_attempt_soal ON jawabans (attempt_id, soal_id)`)

	log.Println("Database migrated successfully")
}
