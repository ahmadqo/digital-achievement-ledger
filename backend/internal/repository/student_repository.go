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
	"github.com/ahmadqo/digital-achievement-ledger/internal/response"
)

type StudentRepository interface {
	FindAll(ctx context.Context, filter model.StudentFilter) ([]*model.Student, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Student, error)
	FindByNISN(ctx context.Context, nisn string) (*model.Student, error)
	Create(ctx context.Context, student *model.Student) error
	Update(ctx context.Context, student *model.Student) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdatePhoto(ctx context.Context, id uuid.UUID, photoURL string) error
}

type studentRepository struct {
	db *sqlx.DB
}

func NewStudentRepository(db *sqlx.DB) StudentRepository {
	return &studentRepository{db: db}
}

func (r *studentRepository) FindAll(ctx context.Context, filter model.StudentFilter) ([]*model.Student, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = 10
	}

	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(full_name ILIKE $%d OR nisn ILIKE $%d)", argIdx, argIdx+1))
		search := "%" + filter.Search + "%"
		args = append(args, search, search)
		argIdx += 2
	}

	if filter.Class != "" {
		conditions = append(conditions, fmt.Sprintf("class = $%d", argIdx))
		args = append(args, filter.Class)
		argIdx++
	}

	if filter.YearGraduate != nil {
		conditions = append(conditions, fmt.Sprintf("year_graduate = $%d", argIdx))
		args = append(args, *filter.YearGraduate)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM students WHERE %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch data
	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`
		SELECT id, nisn, full_name, birth_place, birth_date, gender, class,
		       year_entry, year_graduate, photo_url, created_at, updated_at
		FROM students
		WHERE %s
		ORDER BY full_name ASC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, filter.PerPage, offset)

	var students []*model.Student
	if err := r.db.SelectContext(ctx, &students, query, args...); err != nil {
		return nil, 0, err
	}

	return students, total, nil
}

func (r *studentRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Student, error) {
	var student model.Student
	query := `
		SELECT id, nisn, full_name, birth_place, birth_date, gender, class,
		       year_entry, year_graduate, photo_url, created_at, updated_at
		FROM students WHERE id = $1
	`
	err := r.db.GetContext(ctx, &student, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &student, nil
}

func (r *studentRepository) FindByNISN(ctx context.Context, nisn string) (*model.Student, error) {
	var student model.Student
	err := r.db.GetContext(ctx, &student, "SELECT * FROM students WHERE nisn = $1", nisn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &student, nil
}

func (r *studentRepository) Create(ctx context.Context, student *model.Student) error {
	query := `
		INSERT INTO students (id, nisn, full_name, birth_place, birth_date, gender, class,
		                      year_entry, year_graduate, photo_url, created_at, updated_at)
		VALUES (:id, :nisn, :full_name, :birth_place, :birth_date, :gender, :class,
		        :year_entry, :year_graduate, :photo_url, NOW(), NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, student)
	return err
}

func (r *studentRepository) Update(ctx context.Context, student *model.Student) error {
	query := `
		UPDATE students SET
			full_name = :full_name, birth_place = :birth_place, birth_date = :birth_date,
			gender = :gender, class = :class, year_entry = :year_entry,
			year_graduate = :year_graduate, updated_at = NOW()
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, student)
	return err
}

func (r *studentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM students WHERE id = $1", id)
	return err
}

func (r *studentRepository) UpdatePhoto(ctx context.Context, id uuid.UUID, photoURL string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE students SET photo_url = $1, updated_at = $2 WHERE id = $3",
		photoURL, time.Now(), id,
	)
	return err
}

// Pastikan interface terpenuhi
var _ StudentRepository = (*studentRepository)(nil)
var _ = response.Success // suppress unused import