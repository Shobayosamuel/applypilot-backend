package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// Get returns the stored profile JSON, or {} when the user has none yet.
func (r *Repository) Get(ctx context.Context, userID string) (json.RawMessage, time.Time, error) {
	var data []byte
	var at time.Time
	err := r.db.QueryRowContext(ctx, `SELECT data, updated_at FROM profiles WHERE user_id = $1`, userID).Scan(&data, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return json.RawMessage(`{}`), time.Time{}, nil
	}
	return data, at, err
}

func (r *Repository) Upsert(ctx context.Context, userID string, data json.RawMessage) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO profiles (user_id, data, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (user_id) DO UPDATE SET data = EXCLUDED.data, updated_at = now()
		RETURNING updated_at`, userID, []byte(data)).Scan(&at)
	return at, err
}

// InsertIfMissing writes data only when the user has no profile yet.
func (r *Repository) InsertIfMissing(ctx context.Context, userID string, data json.RawMessage) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO profiles (user_id, data) VALUES ($1, $2) ON CONFLICT (user_id) DO NOTHING`, userID, []byte(data))
	return err
}
