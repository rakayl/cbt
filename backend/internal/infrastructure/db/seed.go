package db

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) {
	log.Println("🌱 Seeding database...")

	// Cek apakah sudah ada data
	var count int64
	db.Table("users").Count(&count)
	if count > 0 {
		log.Println("⚠️ Seeder skipped (data already exists)")
		return
	}

	// ── 1. USER (ADMIN) ─────────────────
	adminID := uuid.New()
	err := db.Exec(`
		INSERT INTO users (id, name, email, role, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, adminID, "Admin", "admin@test.com", "admin", time.Now()).Error
	if err != nil {
		log.Println("❌ seed users:", err)
		return
	}

	// ── 2. MAPEL ────────────────────────
	mapelID := uuid.New()
	err = db.Exec(`
		INSERT INTO mapels (id, nama, created_at)
		VALUES (?, ?, ?)
	`, mapelID, "Matematika", time.Now()).Error
	if err != nil {
		log.Println("❌ seed mapel:", err)
		return
	}

	// ── 3. UJIAN ────────────────────────
	ujianID := uuid.New()
	err = db.Exec(`
		INSERT INTO ujians (id, judul, mapel_id, durasi_menit, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, ujianID, "Ujian Matematika", mapelID, 60, "published", time.Now()).Error
	if err != nil {
		log.Println("❌ seed ujian:", err)
		return
	}

	log.Println("✅ Seeder success")
}