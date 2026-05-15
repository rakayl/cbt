package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ExportReportRequest struct {
	ReportType string    `json:"report_type" validate:"required,oneof=tenant_analytics lecturer_report transcript"`
	Format     string    `json:"format" validate:"required,oneof=pdf excel csv"`
	ExamID     uuid.UUID `json:"exam_id"`
}

type ReportFile struct {
	Filename    string
	ContentType string
	Bytes       []byte
}

type reportRow struct {
	Label string
	Value string
}

func (s *service) Export(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, req ExportReportRequest) (ReportFile, error) {
	if err := validate.Struct(req); err != nil {
		return ReportFile{}, err
	}
	rows, err := s.reportRows(ctx, tenantID, actorUserID, permissions, req)
	if err != nil {
		return ReportFile{}, err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	name := fmt.Sprintf("%s-%s", req.ReportType, stamp)
	if req.Format == "pdf" {
		file := ReportFile{Filename: name + ".pdf", ContentType: "application/pdf", Bytes: buildPDF("Enterprise CBT "+title(req.ReportType), rows)}
		s.publishReportEvent(ctx, tenantID, req, file)
		return file, nil
	}
	file := ReportFile{Filename: name + ".csv", ContentType: "text/csv; charset=utf-8", Bytes: buildCSV(rows)}
	s.publishReportEvent(ctx, tenantID, req, file)
	return file, nil
}

func (s *service) publishReportEvent(ctx context.Context, tenantID uuid.UUID, req ExportReportRequest, file ReportFile) {
	if s.deps.Rabbit == nil {
		return
	}
	body := []byte(fmt.Sprintf(`{"tenant_id":"%s","report_type":"%s","format":"%s","filename":"%s"}`, tenantID, req.ReportType, req.Format, file.Filename))
	_ = s.deps.Rabbit.Publish(ctx, "report_queue", body)
}

func (s *service) reportRows(ctx context.Context, tenantID, actorUserID uuid.UUID, permissions []string, req ExportReportRequest) ([]reportRow, error) {
	rows := []reportRow{
		{Label: "Tenant ID", Value: tenantID.String()},
		{Label: "Report Type", Value: title(req.ReportType)},
		{Label: "Generated At", Value: time.Now().UTC().Format(time.RFC3339)},
	}

	var totalSessions, completedSessions, activeSessions, totalAnswers, suspiciousEvents int
	if canManageAllReports(permissions) {
		err := s.deps.DB.QueryRow(ctx, `
			SELECT
				(SELECT count(*) FROM exam_sessions WHERE tenant_id=$1 AND deleted_at IS NULL),
				(SELECT count(*) FROM exam_sessions WHERE tenant_id=$1 AND deleted_at IS NULL AND status_enum='completed'),
				(SELECT count(*) FROM exam_sessions WHERE tenant_id=$1 AND deleted_at IS NULL AND status_enum IN ('started','reconnecting')),
				(SELECT count(*) FROM answers WHERE tenant_id=$1 AND deleted_at IS NULL),
				(SELECT count(*) FROM proctoring_logs WHERE tenant_id=$1 AND deleted_at IS NULL)`,
			tenantID).Scan(&totalSessions, &completedSessions, &activeSessions, &totalAnswers, &suspiciousEvents)
		if err != nil {
			return nil, err
		}
	} else {
		err := s.deps.DB.QueryRow(ctx, `
			SELECT
				(SELECT count(*)
				 FROM exam_sessions es JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
				 WHERE es.tenant_id=$1 AND es.deleted_at IS NULL AND e.owner_user_id=$2),
				(SELECT count(*)
				 FROM exam_sessions es JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
				 WHERE es.tenant_id=$1 AND es.deleted_at IS NULL AND es.status_enum='completed' AND e.owner_user_id=$2),
				(SELECT count(*)
				 FROM exam_sessions es JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
				 WHERE es.tenant_id=$1 AND es.deleted_at IS NULL AND es.status_enum IN ('started','reconnecting') AND e.owner_user_id=$2),
				(SELECT count(*)
				 FROM answers a JOIN exam_sessions es ON es.id=a.exam_session_id AND es.deleted_at IS NULL JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
				 WHERE a.tenant_id=$1 AND a.deleted_at IS NULL AND e.owner_user_id=$2),
				(SELECT count(*)
				 FROM proctoring_logs pl JOIN exam_sessions es ON es.id=pl.exam_session_id AND es.deleted_at IS NULL JOIN exams e ON e.id=es.exam_id AND e.deleted_at IS NULL
				 WHERE pl.tenant_id=$1 AND pl.deleted_at IS NULL AND e.owner_user_id=$2)`,
			tenantID, actorUserID).Scan(&totalSessions, &completedSessions, &activeSessions, &totalAnswers, &suspiciousEvents)
		if err != nil {
			return nil, err
		}
	}
	rows = append(rows,
		reportRow{Label: "Total Exam Sessions", Value: strconv.Itoa(totalSessions)},
		reportRow{Label: "Completed Sessions", Value: strconv.Itoa(completedSessions)},
		reportRow{Label: "Active Sessions", Value: strconv.Itoa(activeSessions)},
		reportRow{Label: "Total Answers", Value: strconv.Itoa(totalAnswers)},
		reportRow{Label: "Suspicious Events", Value: strconv.Itoa(suspiciousEvents)},
	)

	if req.ReportType == "tenant_analytics" {
		var students, lecturers, courses, exams int
		err := s.deps.DB.QueryRow(ctx, `
			SELECT
				(SELECT count(*) FROM students WHERE tenant_id=$1 AND deleted_at IS NULL),
				(SELECT count(*) FROM lecturers WHERE tenant_id=$1 AND deleted_at IS NULL),
				(SELECT count(*) FROM courses WHERE tenant_id=$1 AND deleted_at IS NULL),
				(SELECT count(*) FROM exams WHERE tenant_id=$1 AND deleted_at IS NULL)`,
			tenantID).Scan(&students, &lecturers, &courses, &exams)
		if err != nil {
			return nil, err
		}
		rows = append(rows,
			reportRow{Label: "Students", Value: strconv.Itoa(students)},
			reportRow{Label: "Lecturers", Value: strconv.Itoa(lecturers)},
			reportRow{Label: "Courses", Value: strconv.Itoa(courses)},
			reportRow{Label: "Exams", Value: strconv.Itoa(exams)},
		)
	}
	return rows, nil
}

func canManageAllReports(permissions []string) bool {
	for _, permission := range permissions {
		if permission == "*" || permission == "users:read" || permission == "tenants:read" {
			return true
		}
	}
	return false
}

func buildCSV(rows []reportRow) []byte {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Metric", "Value"})
	for _, row := range rows {
		_ = w.Write([]string{row.Label, row.Value})
	}
	w.Flush()
	return buf.Bytes()
}

func buildPDF(reportTitle string, rows []reportRow) []byte {
	lines := []string{reportTitle}
	for _, row := range rows {
		lines = append(lines, row.Label+": "+row.Value)
	}
	content := "BT\n/F1 14 Tf\n50 790 Td\n"
	for i, line := range lines {
		if i == 1 {
			content += "/F1 10 Tf\n0 -28 Td\n"
		} else if i > 1 {
			content += "0 -16 Td\n"
		}
		content += "(" + pdfEscape(line) + ") Tj\n"
	}
	content += "ET"

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects)+1)
	offsets = append(offsets, 0)
	for i, obj := range objects {
		offsets = append(offsets, buf.Len())
		buf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, obj))
	}
	xref := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(objects)+1))
	for i := 1; i < len(offsets); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref))
	return buf.Bytes()
}

func pdfEscape(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", "\\(")
	value = strings.ReplaceAll(value, ")", "\\)")
	return value
}

func title(value string) string {
	return strings.Title(strings.ReplaceAll(value, "_", " "))
}
