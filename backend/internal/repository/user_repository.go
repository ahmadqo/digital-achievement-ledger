package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ahmadqo/digital-achievement-ledger/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
}

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `
		SELECT id, name, email, password, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = TRUE
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found, bukan error
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	query := `
		SELECT id, name, email, password, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, name, email, password, role, is_active, created_at, updated_at)
		VALUES (:id, :name, :email, :password, :role, :is_active, NOW(), NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET name = :name, email = :email, role = :role, is_active = :is_active, updated_at = NOW()
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}