package ai

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/database"
)

// Generation is one logged Claude call.
type Generation struct {
	ID           string
	UserID       string
	JobID        *string
	Kind         string
	InputSummary string
	Output       string
	Model        string
	InputTokens  int
	OutputTokens int
	CacheRead    int
	CreatedAt    time.Time
}

type Usage struct {
	CallsToday, Calls30d                   int
	InputTokens, OutputTokens, CacheTokens int
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// Log records a generation and returns its id.
func (r *Repository) Log(ctx context.Context, g *Generation) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO ai_generations (user_id, job_id, kind, input_summary, output, model, input_tokens, output_tokens, cache_read_tokens)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		g.UserID, g.JobID, g.Kind, g.InputSummary, g.Output, g.Model, g.InputTokens, g.OutputTokens, g.CacheRead).Scan(&id)
	return id, err
}

var errGenerationNotFound = errors.New("generation not found")

// Get returns one generation owned by userID, or errGenerationNotFound.
func (r *Repository) Get(ctx context.Context, userID, id string) (*Generation, error) {
	var g Generation
	var job sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id, user_id, job_id, kind, input_summary, output, model, input_tokens, output_tokens, cache_read_tokens, created_at
		FROM ai_generations WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&g.ID, &g.UserID, &job, &g.Kind, &g.InputSummary, &g.Output, &g.Model, &g.InputTokens, &g.OutputTokens, &g.CacheRead, &g.CreatedAt)
	if database.IsNotFound(err) {
		return nil, errGenerationNotFound
	}
	if job.Valid {
		g.JobID = &job.String
	}
	return &g, err
}

func (r *Repository) CallsToday(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM ai_generations WHERE user_id = $1 AND created_at > now() - interval '24 hours'`, userID).Scan(&n)
	return n, err
}

func (r *Repository) Usage(ctx context.Context, userID string) (*Usage, error) {
	u := &Usage{}
	err := r.db.QueryRowContext(ctx, `SELECT
		count(*) FILTER (WHERE created_at > now() - interval '24 hours'),
		count(*),
		COALESCE(sum(input_tokens), 0), COALESCE(sum(output_tokens), 0), COALESCE(sum(cache_read_tokens), 0)
		FROM ai_generations WHERE user_id = $1 AND created_at > now() - interval '30 days'`, userID).
		Scan(&u.CallsToday, &u.Calls30d, &u.InputTokens, &u.OutputTokens, &u.CacheTokens)
	return u, err
}

func (r *Repository) List(ctx context.Context, userID, jobID, kind string) ([]Generation, error) {
	q := `SELECT id, user_id, job_id, kind, input_summary, output, model, input_tokens, output_tokens, cache_read_tokens, created_at
		FROM ai_generations WHERE user_id = $1`
	args := []any{userID}
	if jobID != "" {
		args = append(args, jobID)
		q += ` AND job_id::text = $` + strconv.Itoa(len(args))
	}
	if kind != "" {
		args = append(args, kind)
		q += ` AND kind = $` + strconv.Itoa(len(args))
	}
	rows, err := r.db.QueryContext(ctx, q+` ORDER BY created_at DESC LIMIT 50`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Generation{}
	for rows.Next() {
		var g Generation
		var job sql.NullString
		if err := rows.Scan(&g.ID, &g.UserID, &job, &g.Kind, &g.InputSummary, &g.Output, &g.Model, &g.InputTokens, &g.OutputTokens, &g.CacheRead, &g.CreatedAt); err != nil {
			return nil, err
		}
		if job.Valid {
			g.JobID = &job.String
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
