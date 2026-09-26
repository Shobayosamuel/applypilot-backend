package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/database"
)

// User is the persisted account.
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
}

var (
	errNotFound  = errors.New("user not found")
	errDuplicate = errors.New("email already registered")
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, email, name, hash string) (*User, error) {
	u := &User{Email: email, Name: name, PasswordHash: hash}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING id, created_at`,
		email, name, hash).Scan(&u.ID, &u.CreatedAt)
	if database.IsUniqueViolation(err) {
		return nil, errDuplicate
	}
	return u, err
}

func (r *Repository) ByEmail(ctx context.Context, email string) (*User, error) {
	return r.one(ctx, `WHERE lower(email) = lower($1)`, email)
}

func (r *Repository) ByID(ctx context.Context, id string) (*User, error) {
	return r.one(ctx, `WHERE id = $1`, id)
}

func (r *Repository) one(ctx context.Context, where string, arg any) (*User, error) {
	u := &User{}
	err := r.db.QueryRowContext(ctx, `SELECT id, email, name, password_hash, created_at FROM users `+where, arg).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	if database.IsNotFound(err) {
		return nil, errNotFound
	}
	return u, err
}
