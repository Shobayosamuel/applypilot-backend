package resume

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/database"
)

type Resume struct {
	ID          string
	UserID      string
	Name        string
	Filename    string
	ContentType string
	Size        int
	IsDefault   bool
	CreatedAt   time.Time
}

var errNotFound = errors.New("resume not found")

const cols = `id, user_id, name, filename, content_type, size_bytes, is_default, created_at`

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func scan(sc interface{ Scan(...any) error }, extra ...any) (*Resume, error) {
	var r Resume
	dest := append([]any{&r.ID, &r.UserID, &r.Name, &r.Filename, &r.ContentType, &r.Size, &r.IsDefault, &r.CreatedAt}, extra...)
	if err := sc.Scan(dest...); err != nil {
		return nil, err
	}
	return &r, nil
}

// Create stores a file; the user's first resume becomes their default.
func (r *Repository) Create(ctx context.Context, userID, name, filename, contentType string, data []byte) (*Resume, error) {
	var out *Resume
	err := database.Tx(ctx, r.db, func(tx *sql.Tx) error {
		var hasDefault bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM resumes WHERE user_id = $1 AND is_default)`, userID).Scan(&hasDefault); err != nil {
			return err
		}
		var err error
		out, err = scan(tx.QueryRowContext(ctx, `
			INSERT INTO resumes (user_id, name, filename, content_type, size_bytes, data, is_default)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+cols,
			userID, name, filename, contentType, len(data), data, !hasDefault))
		return err
	})
	return out, err
}

func (r *Repository) List(ctx context.Context, userID string) ([]Resume, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+cols+` FROM resumes WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Resume{}
	for rows.Next() {
		res, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *res)
	}
	return out, rows.Err()
}

// File returns the resume and its bytes. id "default" picks the default (or newest) resume.
func (r *Repository) File(ctx context.Context, userID, id string) (*Resume, []byte, error) {
	q := `SELECT ` + cols + `, data FROM resumes WHERE user_id = $1 AND id = $2`
	args := []any{userID, id}
	if id == "default" {
		q = `SELECT ` + cols + `, data FROM resumes WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC LIMIT 1`
		args = args[:1]
	}
	var data []byte
	res, err := scan(r.db.QueryRowContext(ctx, q, args...), &data)
	if database.IsNotFound(err) {
		return nil, nil, errNotFound
	}
	return res, data, err
}

func (r *Repository) SetDefault(ctx context.Context, userID, id string) error {
	return database.Tx(ctx, r.db, func(tx *sql.Tx) error {
		var exists bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM resumes WHERE user_id = $1 AND id = $2)`, userID, id).Scan(&exists)
		if database.IsInvalidInput(err) || (err == nil && !exists) {
			return errNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE resumes SET is_default = false WHERE user_id = $1 AND is_default`, userID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE resumes SET is_default = true WHERE user_id = $1 AND id = $2`, userID, id)
		return err
	})
}

// Delete removes a resume; if it was the default, the newest remaining one takes over.
func (r *Repository) Delete(ctx context.Context, userID, id string) error {
	return database.Tx(ctx, r.db, func(tx *sql.Tx) error {
		var wasDefault bool
		err := tx.QueryRowContext(ctx, `DELETE FROM resumes WHERE user_id = $1 AND id = $2 RETURNING is_default`, userID, id).Scan(&wasDefault)
		if database.IsNotFound(err) {
			return errNotFound
		} else if err != nil {
			return err
		}
		if wasDefault {
			_, err = tx.ExecContext(ctx, `UPDATE resumes SET is_default = true WHERE id = (
				SELECT id FROM resumes WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1)`, userID)
		}
		return err
	})
}
