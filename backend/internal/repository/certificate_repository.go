package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
)

type CertificateRepository interface {
	FindAll(ctx context.Context, filter model.CertificateFilter) ([]*model.Certificate, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Certificate, error)
	FindByIDWithDetail(ctx context.Context, id uuid.UUID) (*model.CertificateDetail, error)
	FindByQRToken(ctx context.Context, token string) (*model.Certificate, error)
	Create(ctx context.Context, cert *model.Certificate, achievementIDs []uuid.UUID) error
	UpdatePDFURL(ctx context.Context, id uuid.UUID, pdfURL string) error
	Revoke(ctx context.Context, id uuid.UUID) error
	CountByYear(ctx context.Context, year int) (int, error)
}

type certificateRepository struct {
	db *sqlx.DB
}

func NewCertificateRepository(db *sqlx.DB) CertificateRepository {
	return &certificateRepository{db: db}
}

func (r *certificateRepository) FindAll(ctx context.Context, filter model.CertificateFilter) ([]*model.Certificate, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 10
	}

	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.StudentID != "" {
		conditions = append(conditions, fmt.Sprintf("c.student_id = $%d", argIdx))
		args = append(args, filter.StudentID)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM certificates c WHERE %s", where), args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`
		SELECT c.*, s.full_name as student_name, s.nisn as student_nisn,
		       u.name as issued_by_name
		FROM certificates c
		LEFT JOIN students s ON c.student_id = s.id
		LEFT JOIN users u ON c.issued_by = u.id
		WHERE %s
		ORDER BY c.issued_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	var certs []*model.Certificate
	if err := r.db.SelectContext(ctx, &certs, query, args...); err != nil {
		return nil, 0, err
	}

	return certs, total, nil
}

func (r *certificateRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Certificate, error) {
	var cert model.Certificate
	query := `
		SELECT c.*, s.full_name as student_name, s.nisn as student_nisn,
		       u.name as issued_by_name
		FROM certificates c
		LEFT JOIN students s ON c.student_id = s.id
		LEFT JOIN users u ON c.issued_by = u.id
		WHERE c.id = $1
	`
	err := r.db.GetContext(ctx, &cert, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &cert, nil
}

func (r *certificateRepository) FindByIDWithDetail(ctx context.Context, id uuid.UUID) (*model.CertificateDetail, error) {
	cert, err := r.FindByID(ctx, id)
	if err != nil || cert == nil {
		return nil, err
	}

	// Ambil data student
	var student model.Student
	if err := r.db.GetContext(ctx, &student,
		"SELECT * FROM students WHERE id = $1", cert.StudentID); err != nil {
		return nil, err
	}

	// Ambil achievements yang terkait sertifikat ini
	var achievements []model.Achievement
	achievementQuery := `
		SELECT a.*, ac.name as category_name, cl.name as level_name
		FROM achievements a
		JOIN certificate_achievements ca ON a.id = ca.achievement_id
		LEFT JOIN achievement_categories ac ON a.category_id = ac.id
		LEFT JOIN competition_levels cl ON a.level_id = cl.id
		WHERE ca.certificate_id = $1
		ORDER BY a.year DESC
	`
	if err := r.db.SelectContext(ctx, &achievements, achievementQuery, id); err != nil {
		return nil, err
	}

	// Ambil attachments tiap achievement
	achievementsWithAtt := make([]model.AchievementWithAttachments, len(achievements))
	for i, a := range achievements {
		var attachments []model.AchievementAttachment
		r.db.SelectContext(ctx, &attachments,
			"SELECT * FROM achievement_attachments WHERE achievement_id = $1", a.ID)
		achievementsWithAtt[i] = model.AchievementWithAttachments{
			Achievement: a,
			Attachments: attachments,
		}
	}

	return &model.CertificateDetail{
		Certificate:  *cert,
		Student:      &student,
		Achievements: achievementsWithAtt,
	}, nil
}

func (r *certificateRepository) FindByQRToken(ctx context.Context, token string) (*model.Certificate, error) {
	var cert model.Certificate
	query := `
		SELECT c.*, s.full_name as student_name, s.nisn as student_nisn
		FROM certificates c
		LEFT JOIN students s ON c.student_id = s.id
		WHERE c.qr_token = $1
	`
	err := r.db.GetContext(ctx, &cert, query, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &cert, nil
}

func (r *certificateRepository) Create(ctx context.Context, cert *model.Certificate, achievementIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert certificate
	query := `
		INSERT INTO certificates (id, student_id, certificate_number, issued_at, issued_by,
		                          valid_until, qr_token, status, notes, created_at)
		VALUES (:id, :student_id, :certificate_number, :issued_at, :issued_by,
		        :valid_until, :qr_token, :status, :notes, NOW())
	`
	if _, err := tx.NamedExecContext(ctx, query, cert); err != nil {
		return err
	}

	// Insert relasi certificate_achievements
	for _, achID := range achievementIDs {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO certificate_achievements (certificate_id, achievement_id) VALUES ($1, $2)",
			cert.ID, achID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *certificateRepository) UpdatePDFURL(ctx context.Context, id uuid.UUID, pdfURL string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE certificates SET pdf_url = $1 WHERE id = $2", pdfURL, id)
	return err
}

func (r *certificateRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE certificates SET status = 'revoked', updated_at = $1 WHERE id = $2",
		time.Now(), id)
	return err
}

func (r *certificateRepository) CountByYear(ctx context.Context, year int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM certificates WHERE EXTRACT(YEAR FROM issued_at) = $1", year,
	).Scan(&count)
	return count, err
}