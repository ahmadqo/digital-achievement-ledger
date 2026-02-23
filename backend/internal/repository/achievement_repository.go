package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
)

type AchievementRepository interface {
	FindAll(ctx context.Context, filter model.AchievementFilter) ([]*model.Achievement, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Achievement, error)
	FindByIDWithAttachments(ctx context.Context, id uuid.UUID) (*model.AchievementWithAttachments, error)
	FindByStudentID(ctx context.Context, studentID uuid.UUID) ([]*model.Achievement, error)
	Create(ctx context.Context, achievement *model.Achievement) error
	Update(ctx context.Context, achievement *model.Achievement) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Attachments
	AddAttachment(ctx context.Context, att *model.AchievementAttachment) error
	DeleteAttachment(ctx context.Context, id uuid.UUID) (*model.AchievementAttachment, error)
	FindAttachmentByID(ctx context.Context, id uuid.UUID) (*model.AchievementAttachment, error)

	// References
	FindAllCategories(ctx context.Context) ([]*model.AchievementCategory, error)
	FindAllLevels(ctx context.Context) ([]*model.CompetitionLevel, error)
}

type achievementRepository struct {
	db *sqlx.DB
}

func NewAchievementRepository(db *sqlx.DB) AchievementRepository {
	return &achievementRepository{db: db}
}

func (r *achievementRepository) FindAll(ctx context.Context, filter model.AchievementFilter) ([]*model.Achievement, int64, error) {
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
		conditions = append(conditions, fmt.Sprintf("a.student_id = $%d", argIdx))
		args = append(args, filter.StudentID)
		argIdx++
	}
	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("a.category_id = $%d", argIdx))
		args = append(args, *filter.CategoryID)
		argIdx++
	}
	if filter.LevelID != nil {
		conditions = append(conditions, fmt.Sprintf("a.level_id = $%d", argIdx))
		args = append(args, *filter.LevelID)
		argIdx++
	}
	if filter.Year != nil {
		conditions = append(conditions, fmt.Sprintf("a.year = $%d", argIdx))
		args = append(args, *filter.Year)
		argIdx++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(a.competition_name ILIKE $%d OR a.organizer ILIKE $%d)", argIdx, argIdx+1))
		search := "%" + filter.Search + "%"
		args = append(args, search, search)
		argIdx += 2
	}

	where := strings.Join(conditions, " AND ")

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM achievements a WHERE %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`
		SELECT a.*, ac.name as category_name, cl.name as level_name,
		       s.full_name as student_name, s.nisn as student_nisn
		FROM achievements a
		LEFT JOIN achievement_categories ac ON a.category_id = ac.id
		LEFT JOIN competition_levels cl ON a.level_id = cl.id
		LEFT JOIN students s ON a.student_id = s.id
		WHERE %s
		ORDER BY a.year DESC, a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	var achievements []*model.Achievement
	if err := r.db.SelectContext(ctx, &achievements, query, args...); err != nil {
		return nil, 0, err
	}

	return achievements, total, nil
}

func (r *achievementRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Achievement, error) {
	var a model.Achievement
	query := `
		SELECT a.*, ac.name as category_name, cl.name as level_name,
		       s.full_name as student_name, s.nisn as student_nisn
		FROM achievements a
		LEFT JOIN achievement_categories ac ON a.category_id = ac.id
		LEFT JOIN competition_levels cl ON a.level_id = cl.id
		LEFT JOIN students s ON a.student_id = s.id
		WHERE a.id = $1
	`
	err := r.db.GetContext(ctx, &a, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *achievementRepository) FindByIDWithAttachments(ctx context.Context, id uuid.UUID) (*model.AchievementWithAttachments, error) {
	achievement, err := r.FindByID(ctx, id)
	if err != nil || achievement == nil {
		return nil, err
	}

	attachments, err := r.findAttachmentsByAchievementID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.AchievementWithAttachments{
		Achievement: *achievement,
		Attachments: attachments,
	}, nil
}

func (r *achievementRepository) FindByStudentID(ctx context.Context, studentID uuid.UUID) ([]*model.Achievement, error) {
	var achievements []*model.Achievement
	query := `
		SELECT a.*, ac.name as category_name, cl.name as level_name
		FROM achievements a
		LEFT JOIN achievement_categories ac ON a.category_id = ac.id
		LEFT JOIN competition_levels cl ON a.level_id = cl.id
		WHERE a.student_id = $1
		ORDER BY a.year DESC
	`
	if err := r.db.SelectContext(ctx, &achievements, query, studentID); err != nil {
		return nil, err
	}
	return achievements, nil
}

func (r *achievementRepository) Create(ctx context.Context, achievement *model.Achievement) error {
	query := `
		INSERT INTO achievements (id, student_id, competition_name, organizer, category_id,
		                          rank, level_id, year, description, created_by, created_at, updated_at)
		VALUES (:id, :student_id, :competition_name, :organizer, :category_id,
		        :rank, :level_id, :year, :description, :created_by, NOW(), NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, achievement)
	return err
}

func (r *achievementRepository) Update(ctx context.Context, achievement *model.Achievement) error {
	query := `
		UPDATE achievements SET
			competition_name = :competition_name, organizer = :organizer,
			category_id = :category_id, rank = :rank, level_id = :level_id,
			year = :year, description = :description, updated_at = NOW()
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, achievement)
	return err
}

func (r *achievementRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM achievements WHERE id = $1", id)
	return err
}

func (r *achievementRepository) AddAttachment(ctx context.Context, att *model.AchievementAttachment) error {
	query := `
		INSERT INTO achievement_attachments (id, achievement_id, file_url, file_name, file_type, label, uploaded_at)
		VALUES (:id, :achievement_id, :file_url, :file_name, :file_type, :label, NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, att)
	return err
}

func (r *achievementRepository) FindAttachmentByID(ctx context.Context, id uuid.UUID) (*model.AchievementAttachment, error) {
	var att model.AchievementAttachment
	err := r.db.GetContext(ctx, &att, "SELECT * FROM achievement_attachments WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &att, nil
}

func (r *achievementRepository) DeleteAttachment(ctx context.Context, id uuid.UUID) (*model.AchievementAttachment, error) {
	att, err := r.FindAttachmentByID(ctx, id)
	if err != nil || att == nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, "DELETE FROM achievement_attachments WHERE id = $1", id)
	return att, err
}

func (r *achievementRepository) findAttachmentsByAchievementID(ctx context.Context, achievementID uuid.UUID) ([]model.AchievementAttachment, error) {
	var attachments []model.AchievementAttachment
	err := r.db.SelectContext(ctx, &attachments,
		"SELECT * FROM achievement_attachments WHERE achievement_id = $1 ORDER BY uploaded_at ASC",
		achievementID,
	)
	return attachments, err
}

func (r *achievementRepository) FindAllCategories(ctx context.Context) ([]*model.AchievementCategory, error) {
	var categories []*model.AchievementCategory
	err := r.db.SelectContext(ctx, &categories, "SELECT * FROM achievement_categories ORDER BY id")
	return categories, err
}

func (r *achievementRepository) FindAllLevels(ctx context.Context) ([]*model.CompetitionLevel, error) {
	var levels []*model.CompetitionLevel
	err := r.db.SelectContext(ctx, &levels, "SELECT * FROM competition_levels ORDER BY order_rank")
	return levels, err
}