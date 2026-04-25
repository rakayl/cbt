package usecase

import (
	"context"
	"log"
	"time"

	"cbt-enterprise/internal/domain"
	"cbt-enterprise/internal/infrastructure/queue"

	"github.com/google/uuid"
)

type ScoringUsecase struct {
	attemptRepo    domain.AttemptRepository
	ujianRepo      domain.UjianRepository
	penilaianRepo  domain.PenilaianRepository
	soalRepo       domain.SoalRepository
	queue          *queue.ScoringQueue
}

func NewScoringUsecase(
	ar domain.AttemptRepository,
	ur domain.UjianRepository,
	pr domain.PenilaianRepository,
	sr domain.SoalRepository,
	sq *queue.ScoringQueue,
) *ScoringUsecase {
	return &ScoringUsecase{
		attemptRepo:   ar,
		ujianRepo:     ur,
		penilaianRepo: pr,
		soalRepo:      sr,
		queue:         sq,
	}
}

// StartWorker runs a blocking worker loop
func (uc *ScoringUsecase) StartWorker(ctx context.Context) {
	log.Println("[ScoringWorker] Started")
	for {
		select {
		case <-ctx.Done():
			log.Println("[ScoringWorker] Stopping")
			return
		default:
			job, err := uc.queue.Pop(ctx)
			if err != nil {
				continue
			}
			if err := uc.Score(ctx, *job); err != nil {
				log.Printf("[ScoringWorker] Error scoring attempt %s: %v", job.AttemptID, err)
			}
		}
	}
}

func (uc *ScoringUsecase) Score(ctx context.Context, job queue.ScoringJob) error {
	attemptID, err := uuid.Parse(job.AttemptID)
	if err != nil {
		return err
	}

	// Get all jawaban
	jawabans, err := uc.attemptRepo.GetJawaban(ctx, attemptID)
	if err != nil {
		return err
	}

	// Build soal map for correctness check
	soalMap := map[uuid.UUID]*domain.Soal{}
	for _, j := range jawabans {
		if _, ok := soalMap[j.SoalID]; !ok {
			soal, err := uc.soalRepo.FindByID(ctx, j.SoalID)
			if err == nil {
				soalMap[j.SoalID] = soal
			}
		}
	}

	var totalPG, totalEssay float64

	for _, j := range jawabans {
		soal, ok := soalMap[j.SoalID]
		if !ok {
			continue
		}

		if soal.Tipe == domain.SoalPG && j.OpsiID != nil {
			// Auto-score PG
			for _, opsi := range soal.Opsi {
				if opsi.ID == *j.OpsiID && opsi.IsBenar {
					totalPG += float64(soal.Poin)
					break
				}
			}
		}
		// Essay: totalEssay stays 0 until manual grading
	}

	now := time.Now()
	penilaian := &domain.Penilaian{
		ID:         uuid.New(),
		AttemptID:  attemptID,
		Skor:       totalPG + totalEssay,
		NilaiPG:    totalPG,
		NilaiEssay: totalEssay,
		Status:     "selesai",
		GradeAt:    &now,
	}

	// Check if essay exists → keep status pending for manual
	for _, j := range jawabans {
		soal := soalMap[j.SoalID]
		if soal != nil && soal.Tipe == domain.SoalEssay {
			penilaian.Status = "pending_essay"
			break
		}
	}

	return uc.penilaianRepo.Create(ctx, penilaian)
}
