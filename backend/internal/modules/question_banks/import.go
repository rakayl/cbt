package question_banks

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type ImportedQuestion struct {
	Content       string
	AnswerMode    string
	Difficulty    string
	Score         float64
	Options       []ImportedOption
	CorrectLabels map[string]bool
	Explanation   string
	RowNumber     int
}

type ImportedOption struct {
	Label string
	Text  string
}

type ImportResult struct {
	QuestionBankID uuid.UUID `json:"question_bank_id"`
	Imported       int       `json:"imported"`
	Skipped        int       `json:"skipped"`
	SourceType     string    `json:"source_type"`
}

type UploadResult struct {
	ObjectKey string `json:"object_key"`
	URL       string `json:"url"`
}

func (s *service) ImportQuestions(ctx context.Context, tenantID, questionBankID, actorUserID uuid.UUID, permissions []string, requestedLecturerID *uuid.UUID, filename string, payload []byte) (ImportResult, error) {
	questions, sourceType, err := parseQuestions(filename, payload)
	if err != nil {
		return ImportResult{}, err
	}
	if err := s.ensureQuestionBankWritable(ctx, tenantID, questionBankID, actorUserID, permissions); err != nil {
		return ImportResult{}, err
	}
	lecturerID, ownerUserID, err := s.resolveQuestionOwner(ctx, tenantID, actorUserID, permissions, requestedLecturerID)
	if err != nil {
		return ImportResult{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	defer tx.Rollback(ctx)
	imported := 0
	skipped := 0
	for _, item := range questions {
		if strings.TrimSpace(item.Content) == "" {
			skipped++
			continue
		}
		if err := validateImportedQuestion(item); err != nil {
			return ImportResult{}, err
		}
		questionID := uuid.New()
		answerMode := normalizeAnswerMode(item.AnswerMode)
		difficulty := item.Difficulty
		if difficulty == "" {
			difficulty = "medium"
		}
		score := item.Score
		if score <= 0 {
			score = 1
		}
		questionMeta, _ := json.Marshal(map[string]any{
			"answer_mode":   answerMode,
			"question_type": "multiple_choice",
			"imported_from": sourceType,
			"lecturer_id":   lecturerID.String(),
			"owner_user_id": ownerUserID.String(),
		})
		_, err = tx.Exec(ctx, `
			INSERT INTO questions(id,tenant_id,code,name,description,status,metadata,question_bank_id,lecturer_id,owner_user_id,question_type,answer_mode,difficulty,content,explanation,score,version)
			VALUES($1,$2,$3,$4,$5,'active',$6,$7,$8,$9,'multiple_choice',$10,$11,$12,$13,$14,1)`,
			questionID, tenantID, "Q-"+questionID.String()[:8], truncate(item.Content, 120), truncate(item.Content, 160), questionMeta, questionBankID, lecturerID, ownerUserID, answerMode, difficulty, item.Content, item.Explanation, score)
		if err != nil {
			return ImportResult{}, err
		}
		for idx, option := range item.Options {
			optionID := uuid.New()
			label := strings.ToUpper(strings.TrimSpace(option.Label))
			optionMeta, _ := json.Marshal(map[string]any{"label": label, "text": option.Text})
			_, err = tx.Exec(ctx, `
				INSERT INTO question_options(id,tenant_id,code,name,description,status,metadata,question_id,is_correct,sort_order)
				VALUES($1,$2,$3,$4,$5,'active',$6,$7,$8,$9)`,
				optionID, tenantID, label, label, option.Text, optionMeta, questionID, item.CorrectLabels[label], idx+1)
			if err != nil {
				return ImportResult{}, err
			}
		}
		imported++
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportResult{}, err
	}
	if s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "question_import_queue", []byte(questionBankID.String()))
	}
	return ImportResult{QuestionBankID: questionBankID, Imported: imported, Skipped: skipped, SourceType: sourceType}, nil
}

func (s *service) UploadMedia(ctx context.Context, tenantID, questionBankID uuid.UUID, filename string, contentType string, reader io.Reader, size int64) (UploadResult, error) {
	if s.deps.Storage == nil {
		return UploadResult{}, errors.New("object storage is not configured")
	}
	objectKey := fmt.Sprintf("tenants/%s/question-banks/%s/%s-%s", tenantID, questionBankID, uuid.NewString(), filepath.Base(filename))
	_, err := s.deps.Storage.PutObject(ctx, s.deps.Config.S3Bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return UploadResult{}, err
	}
	url, err := s.deps.Storage.PresignedGetObject(ctx, s.deps.Config.S3Bucket, objectKey, 15*time.Minute, nil)
	if err != nil {
		return UploadResult{}, err
	}
	return UploadResult{ObjectKey: objectKey, URL: url.String()}, nil
}

func QuestionImportTemplateCSV() string {
	var out bytes.Buffer
	writer := csv.NewWriter(&out)
	_ = writer.Write([]string{"question_text", "answer_mode", "difficulty", "score", "option_a", "option_b", "option_c", "option_d", "option_e", "option_f", "correct_answers", "explanation"})
	_ = writer.Write([]string{"Ibu kota Indonesia adalah?", "single", "easy", "1", "Bandung", "Jakarta", "Surabaya", "Medan", "Semarang", "", "B", "Jakarta adalah ibu kota Indonesia."})
	_ = writer.Write([]string{"Pilih bilangan prima berikut.", "multiple", "medium", "1", "2", "3", "4", "5", "6", "7", "A,B,D,F", "2, 3, 5, dan 7 adalah bilangan prima."})
	writer.Flush()
	return out.String()
}

func parseQuestions(filename string, payload []byte) ([]ImportedQuestion, string, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".csv":
		return parseCSV(payload)
	case ".xlsx":
		items, err := parseXLSX(payload)
		return items, "xlsx", err
	case ".docx":
		return nil, "", errors.New("docx import is not supported for the simplified multiple choice template; download and use the csv template")
	default:
		return nil, "", errors.New("unsupported import file type")
	}
}

func parseCSV(payload []byte) ([]ImportedQuestion, string, error) {
	payload = bytes.TrimPrefix(payload, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(payload))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, "", err
	}
	out := make([]ImportedQuestion, 0, len(records))
	for i, row := range records {
		if i == 0 && isImportHeader(row) {
			continue
		}
		out = append(out, rowToQuestion(row, i+1))
	}
	return out, "csv", nil
}

func parseXLSX(payload []byte) ([]ImportedQuestion, error) {
	zr, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return nil, err
	}
	shared := []string{}
	for _, file := range zr.File {
		if file.Name == "xl/sharedStrings.xml" {
			texts, err := readSharedStrings(file)
			if err != nil {
				return nil, err
			}
			shared = texts
		}
	}
	for _, file := range zr.File {
		if file.Name == "xl/worksheets/sheet1.xml" {
			rows, err := readSheetRows(file, shared)
			if err != nil {
				return nil, err
			}
			out := []ImportedQuestion{}
			for i, row := range rows {
				if i == 0 && isImportHeader(row) {
					continue
				}
				out = append(out, rowToQuestion(row, i+1))
			}
			return out, nil
		}
	}
	return nil, errors.New("xlsx sheet1.xml not found")
}

func rowToQuestion(row []string, rowNumber int) ImportedQuestion {
	padded := make([]string, 12)
	copy(padded, row)
	score, _ := strconv.ParseFloat(strings.TrimSpace(padded[3]), 64)
	options := make([]ImportedOption, 0, 6)
	for index, text := range padded[4:10] {
		if strings.TrimSpace(text) == "" {
			continue
		}
		options = append(options, ImportedOption{
			Label: string(rune('A' + index)),
			Text:  strings.TrimSpace(text),
		})
	}
	return ImportedQuestion{
		Content:       strings.TrimSpace(padded[0]),
		AnswerMode:    strings.TrimSpace(padded[1]),
		Difficulty:    strings.TrimSpace(padded[2]),
		Score:         score,
		Options:       options,
		CorrectLabels: parseCorrectLabels(padded[10]),
		Explanation:   strings.TrimSpace(padded[11]),
		RowNumber:     rowNumber,
	}
}

func isImportHeader(row []string) bool {
	joined := strings.ToLower(strings.Join(row, ","))
	return strings.Contains(joined, "question_text") || strings.Contains(joined, "correct_answers")
}

func parseCorrectLabels(value string) map[string]bool {
	out := map[string]bool{}
	splitter := func(r rune) bool {
		return r == ',' || r == ';' || r == '|'
	}
	for _, raw := range strings.FieldsFunc(value, splitter) {
		label := strings.ToUpper(strings.TrimSpace(raw))
		if label != "" {
			out[label] = true
		}
	}
	return out
}

func normalizeAnswerMode(value string) string {
	mode := strings.ToLower(strings.TrimSpace(value))
	if mode == "" {
		return "single"
	}
	return mode
}

func validateImportedQuestion(item ImportedQuestion) error {
	mode := normalizeAnswerMode(item.AnswerMode)
	if mode != "single" && mode != "multiple" {
		return fmt.Errorf("row %d: answer_mode must be single or multiple", item.RowNumber)
	}
	if len(item.Options) < 2 {
		return fmt.Errorf("row %d: minimal 2 opsi jawaban", item.RowNumber)
	}
	labels := map[string]bool{}
	correct := 0
	for _, option := range item.Options {
		label := strings.ToUpper(strings.TrimSpace(option.Label))
		if labels[label] {
			return fmt.Errorf("row %d: label opsi duplikat: %s", item.RowNumber, label)
		}
		labels[label] = true
		if strings.TrimSpace(option.Text) == "" {
			return fmt.Errorf("row %d: teks opsi tidak boleh kosong", item.RowNumber)
		}
		if item.CorrectLabels[label] {
			correct++
		}
	}
	for label := range item.CorrectLabels {
		if !labels[label] {
			return fmt.Errorf("row %d: correct_answers berisi label yang tidak ada: %s", item.RowNumber, label)
		}
	}
	if mode == "single" && correct != 1 {
		return fmt.Errorf("row %d: single answer wajib punya tepat 1 jawaban benar", item.RowNumber)
	}
	if mode == "multiple" && correct < 1 {
		return fmt.Errorf("row %d: multiple answer wajib punya minimal 1 jawaban benar", item.RowNumber)
	}
	return nil
}

func (s *service) resolveQuestionOwner(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, requestedLecturerID *uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	if canManageAllQuestionBankOwners(permissions) {
		if requestedLecturerID == nil || *requestedLecturerID == uuid.Nil {
			return uuid.Nil, uuid.Nil, errors.New("admin wajib memilih guru pemilik soal")
		}
		var ownerUserID uuid.UUID
		err := s.deps.DB.QueryRow(ctx, `
			SELECT user_id
			FROM lecturers
			WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status='active'
			LIMIT 1`,
			*requestedLecturerID, tenantID).Scan(&ownerUserID)
		if err != nil {
			return uuid.Nil, uuid.Nil, errors.New("guru tidak ditemukan atau belum punya akun")
		}
		if ownerUserID == uuid.Nil {
			return uuid.Nil, uuid.Nil, errors.New("guru belum terhubung dengan akun user")
		}
		return *requestedLecturerID, ownerUserID, nil
	}

	var lecturerID, ownerUserID uuid.UUID
	err := s.deps.DB.QueryRow(ctx, `
		SELECT id, user_id
		FROM lecturers
		WHERE user_id=$1 AND tenant_id=$2 AND deleted_at IS NULL AND status='active'
		LIMIT 1`,
		actorUserID, tenantID).Scan(&lecturerID, &ownerUserID)
	if err != nil {
		return uuid.Nil, uuid.Nil, errors.New("akun login ini belum terhubung ke data guru")
	}
	if requestedLecturerID != nil && *requestedLecturerID != uuid.Nil && *requestedLecturerID != lecturerID {
		return uuid.Nil, uuid.Nil, errors.New("guru hanya boleh membuat soal atas nama dirinya sendiri")
	}
	return lecturerID, ownerUserID, nil
}

func hasPermission(permissions []string, permission string) bool {
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func readSharedStrings(file *zip.File) ([]string, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var doc struct {
		Texts []string `xml:"si>t"`
	}
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return nil, err
	}
	return doc.Texts, nil
}

func readSheetRows(file *zip.File, shared []string) ([][]string, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var sheet struct {
		Rows []struct {
			Cells []struct {
				Type  string `xml:"t,attr"`
				Value string `xml:"v"`
			} `xml:"c"`
		} `xml:"sheetData>row"`
	}
	if err := xml.NewDecoder(rc).Decode(&sheet); err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		values := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			if cell.Type == "s" {
				idx, _ := strconv.Atoi(cell.Value)
				if idx >= 0 && idx < len(shared) {
					values = append(values, shared[idx])
					continue
				}
			}
			values = append(values, cell.Value)
		}
		rows = append(rows, values)
	}
	return rows, nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
