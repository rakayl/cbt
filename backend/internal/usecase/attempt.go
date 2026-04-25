package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"cbt-enterprise/internal/domain"
	"cbt-enterprise/internal/infrastructure/redis"
	"cbt-enterprise/internal/infrastructure/queue"

	"github.com/google/uuid"
)

const (
	AttemptCacheTTL = 6 * time.Hour
	CheatingThreshold = 5
	CooldownBuffer  = 3
)

type AttemptUsecase struct {
	attemptRepo domain.AttemptRepository
	ujianRepo   domain.UjianRepository
	pesertaRepo domain.PesertaRepository
	actLogRepo  domain.ActivityLogRepository
	redis       *redis.Client
	queue       *queue.ScoringQueue
}

func NewAttemptUsecase(
	ar domain.AttemptRepository,
	ur domain.UjianRepository,
	pr domain.PesertaRepository,
	al domain.ActivityLogRepository,
	rc *redis.Client,
	sq *queue.ScoringQueue,
) *AttemptUsecase {
	return &AttemptUsecase{
		attemptRepo: ar,
		ujianRepo:   ur,
		pesertaRepo: pr,
		actLogRepo:  al,
		redis:       rc,
		queue:       sq,
	}
}

// ─── START EXAM ───────────────────────────────────────────────────────────────

func (uc *AttemptUsecase) StartExam(ctx context.Context, pesertaID, ujianID uuid.UUID) (*domain.Attempt, []domain.AttemptSoal, error) {
	// Check if already has active attempt
	existing, err := uc.attemptRepo.FindActive(ctx, pesertaID, ujianID)
	if err == nil && existing != nil {
		if existing.Status == domain.AttemptOngoing || existing.Status == domain.AttemptPaused {
			soalList, _ := uc.attemptRepo.GetAttemptSoal(ctx, existing.ID)
			return existing, soalList, nil
		}
	}

	ujian, err := uc.ujianRepo.FindByID(ctx, ujianID)
	if err != nil {
		return nil, nil, errors.New("ujian tidak ditemukan")
	}
	if ujian.Status != domain.UjianAktif {
		return nil, nil, errors.New("ujian tidak aktif")
	}

	// Generate deterministic seed
	seed := generateSeed(pesertaID, ujianID)

	attempt := &domain.Attempt{
		ID:        uuid.New(),
		UjianID:   ujianID,
		PesertaID: pesertaID,
		Status:    domain.AttemptOngoing,
		MulaiAt:   time.Now(),
		SisaDetik: ujian.DurasiMenit * 60,
		Seed:      seed,
	}

	if err := uc.attemptRepo.Create(ctx, attempt); err != nil {
		return nil, nil, err
	}

	// Shuffle and assign soal
	soalList, err := uc.ujianRepo.GetSoalList(ctx, ujianID)
	if err != nil {
		return nil, nil, err
	}

	rng := rand.New(rand.NewSource(seed))
	shuffled := shuffleSoal(soalList, rng, ujian.AcakSoal)

	attemptSoal := make([]domain.AttemptSoal, len(shuffled))
	for i, us := range shuffled {
		opsiOrder := ""
		if ujian.AcakOpsi && us.Soal.Tipe == domain.SoalPG {
			opsiOrder = shuffleOpsiOrder(us.Soal.Opsi, rng)
		}
		attemptSoal[i] = domain.AttemptSoal{
			ID:        uuid.New(),
			AttemptID: attempt.ID,
			SoalID:    us.SoalID,
			Urutan:    i + 1,
			OpsiOrder: opsiOrder,
		}
	}

	if err := uc.attemptRepo.SaveAttemptSoal(ctx, attemptSoal); err != nil {
		return nil, nil, err
	}

	// Cache attempt state in Redis
	uc.cacheAttemptState(ctx, attempt.ID, map[string]interface{}{
		"status":    string(attempt.Status),
		"sisa_detik": attempt.SisaDetik,
		"started":   attempt.MulaiAt.Unix(),
	})

	return attempt, attemptSoal, nil
}

// ─── AUTOSAVE JAWABAN ─────────────────────────────────────────────────────────

type SaveJawabanInput struct {
	AttemptID   uuid.UUID
	SoalID      uuid.UUID
	OpsiID      *uuid.UUID
	TeksJawaban string
	SisaDetik   int
}

func (uc *AttemptUsecase) SaveJawaban(ctx context.Context, input SaveJawabanInput) error {
	attempt, err := uc.attemptRepo.FindByID(ctx, input.AttemptID)
	if err != nil {
		return errors.New("attempt tidak ditemukan")
	}
	if attempt.Status != domain.AttemptOngoing {
		return errors.New("ujian tidak sedang berlangsung")
	}

	jawaban := &domain.Jawaban{
		ID:          uuid.New(),
		AttemptID:   input.AttemptID,
		SoalID:      input.SoalID,
		OpsiID:      input.OpsiID,
		TeksJawaban: input.TeksJawaban,
		UpdatedAt:   time.Now(),
	}

	if err := uc.attemptRepo.UpsertJawaban(ctx, jawaban); err != nil {
		return err
	}

	// Update Redis cache
	key := fmt.Sprintf("attempt:%s", input.AttemptID)
	uc.redis.HSet(ctx, key, fmt.Sprintf("soal:%s", input.SoalID), input.TeksJawaban)
	uc.redis.HSet(ctx, key, "sisa_detik", input.SisaDetik)
	uc.redis.Expire(ctx, key, AttemptCacheTTL)

	return nil
}

// ─── FINISH EXAM ─────────────────────────────────────────────────────────────

func (uc *AttemptUsecase) FinishExam(ctx context.Context, attemptID uuid.UUID) error {
	attempt, err := uc.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return err
	}

	now := time.Now()
	attempt.Status = domain.AttemptSelesai
	attempt.SelesaiAt = &now

	if err := uc.attemptRepo.FinishAttempt(ctx, attemptID); err != nil {
		return err
	}

	// Push to scoring queue
	return uc.queue.Push(ctx, queue.ScoringJob{
		AttemptID: attemptID.String(),
		UjianID:   attempt.UjianID.String(),
		PesertaID: attempt.PesertaID.String(),
	})
}

// ─── ANTI-CHEAT: LOG VIOLATION ───────────────────────────────────────────────

type ViolationInput struct {
	AttemptID uuid.UUID
	PesertaID uuid.UUID
	EventType string // tab_switch, fullscreen_exit, multiple_faces, face_mismatch
	Detail    string
}

func (uc *AttemptUsecase) LogViolation(ctx context.Context, input ViolationInput) (bool, error) {
	// Log activity
	uc.actLogRepo.Log(ctx, &domain.ActivityLog{
		AttemptID: input.AttemptID,
		PesertaID: input.PesertaID,
		Event:     input.EventType,
		Detail:    input.Detail,
		CreatedAt: time.Now(),
	})

	// Increment cheating score
	if err := uc.attemptRepo.IncrementCheating(ctx, input.AttemptID); err != nil {
		return false, err
	}

	attempt, err := uc.attemptRepo.FindByID(ctx, input.AttemptID)
	if err != nil {
		return false, err
	}

	// Auto-pause logic
	if attempt.CheatingScore >= CheatingThreshold {
		uc.attemptRepo.PauseAttempt(ctx, input.AttemptID)
		return true, nil // paused
	}

	return false, nil
}

// ─── GET ATTEMPT STATE ────────────────────────────────────────────────────────

func (uc *AttemptUsecase) GetAttemptState(ctx context.Context, attemptID uuid.UUID) (*domain.Attempt, []domain.Jawaban, error) {
	attempt, err := uc.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return nil, nil, err
	}

	jawabans, err := uc.attemptRepo.GetJawaban(ctx, attemptID)
	if err != nil {
		return attempt, nil, nil
	}

	return attempt, jawabans, nil
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func generateSeed(pesertaID, ujianID uuid.UUID) int64 {
	h := int64(0)
	for _, b := range append(pesertaID[:], ujianID[:]...) {
		h = h*31 + int64(b)
	}
	return h
}

func shuffleSoal(list []domain.UjianSoal, rng *rand.Rand, doShuffle bool) []domain.UjianSoal {
	out := make([]domain.UjianSoal, len(list))
	copy(out, list)
	if doShuffle {
		rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	}
	return out
}

func shuffleOpsiOrder(opsi []domain.SoalOpsi, rng *rand.Rand) string {
	ids := make([]string, len(opsi))
	for i, o := range opsi {
		ids[i] = o.ID.String()
	}
	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	b, _ := json.Marshal(ids)
	return string(b)
}

func (uc *AttemptUsecase) cacheAttemptState(ctx context.Context, id uuid.UUID, data map[string]interface{}) {
	key := fmt.Sprintf("attempt:%s", id)
	for k, v := range data {
		uc.redis.HSet(ctx, key, k, v)
	}
	uc.redis.Expire(ctx, key, AttemptCacheTTL)
}
