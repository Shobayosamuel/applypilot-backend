package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/shobayosamuel/applypilot-backend/internal/platform/database"
)

type Job struct {
	ID           string
	UserID       string
	Title        string
	Company      string
	Location     string
	URL          string
	Description  string
	Salary       string
	Source       string
	Status       string
	Notes        string
	MatchScore   *int
	MatchDetails json.RawMessage
	AppliedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var (
	errNotFound  = errors.New("job not found")
	errDuplicate = errors.New("job url already tracked")
)

const cols = `id, user_id, title, company, location, url, description, salary, source, status, notes,
	match_score, match_details, applied_at, created_at, updated_at`

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func scan(sc interface{ Scan(...any) error }, extra ...any) (*Job, error) {
	var j Job
	var md []byte
	var score sql.NullInt64
	var applied sql.NullTime
	dest := append([]any{&j.ID, &j.UserID, &j.Title, &j.Company, &j.Location, &j.URL, &j.Description, &j.Salary, &j.Source,
		&j.Status, &j.Notes, &score, &md, &applied, &j.CreatedAt, &j.UpdatedAt}, extra...)
	if err := sc.Scan(dest...); err != nil {
		return nil, err
	}
	if score.Valid {
		v := int(score.Int64)
		j.MatchScore = &v
	}
	if applied.Valid {
		j.AppliedAt = &applied.Time
	}
	if len(md) > 0 {
		j.MatchDetails = md
	}
	return &j, nil
}

type Filter struct {
	Status string
	Query  string
	Limit  int
}

func (r *Repository) List(ctx context.Context, userID string, f Filter) ([]Job, error) {
	q := `SELECT ` + cols + ` FROM jobs WHERE user_id = $1`
	args := []any{userID}
	arg := func(v any) string { args = append(args, v); return "$" + strconv.Itoa(len(args)) }
	if f.Status != "" {
		q += ` AND status = ` + arg(f.Status)
	}
	if f.Query != "" {
		p := arg("%" + f.Query + "%")
		q += ` AND (title ILIKE ` + p + ` OR company ILIKE ` + p + ` OR location ILIKE ` + p + `)`
	}
	q += ` ORDER BY updated_at DESC LIMIT ` + arg(f.Limit)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Job{}
	for rows.Next() {
		j, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, userID, id string) (*Job, error) {
	j, err := scan(r.db.QueryRowContext(ctx, `SELECT `+cols+` FROM jobs WHERE user_id = $1 AND id = $2`, userID, id))
	if database.IsNotFound(err) {
		return nil, errNotFound
	}
	return j, err
}

func (r *Repository) GetByURL(ctx context.Context, userID, url string) (*Job, error) {
	j, err := scan(r.db.QueryRowContext(ctx, `SELECT `+cols+` FROM jobs WHERE user_id = $1 AND url = $2 AND url <> ''`, userID, url))
	if database.IsNotFound(err) {
		return nil, errNotFound
	}
	return j, err
}

// Upsert inserts a job or, when the URL is already tracked, fills blank fields,
// keeps the longer description and advances status (never back to "saved").
// created reports whether a new row was inserted.
func (r *Repository) Upsert(ctx context.Context, j *Job) (out *Job, created bool, err error) {
	var applied any
	if j.Status == StatusApplied {
		applied = time.Now()
	}
	if j.URL == "" {
		out, err = scan(r.db.QueryRowContext(ctx, `
			INSERT INTO jobs (user_id, title, company, location, url, description, salary, source, status, notes, applied_at)
			VALUES ($1, $2, $3, $4, '', $5, $6, $7, $8, $9, $10) RETURNING `+cols,
			j.UserID, j.Title, j.Company, j.Location, j.Description, j.Salary, j.Source, j.Status, j.Notes, applied))
		return out, true, err
	}
	out, err = scan(r.db.QueryRowContext(ctx, `
		INSERT INTO jobs (user_id, title, company, location, url, description, salary, source, status, notes, applied_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, url) WHERE url <> '' DO UPDATE SET
			title       = CASE WHEN jobs.title = ''    THEN EXCLUDED.title    ELSE jobs.title END,
			company     = CASE WHEN jobs.company = ''  THEN EXCLUDED.company  ELSE jobs.company END,
			location    = CASE WHEN jobs.location = '' THEN EXCLUDED.location ELSE jobs.location END,
			description = CASE WHEN length(EXCLUDED.description) > length(jobs.description) THEN EXCLUDED.description ELSE jobs.description END,
			salary      = CASE WHEN jobs.salary = ''   THEN EXCLUDED.salary   ELSE jobs.salary END,
			status      = CASE WHEN jobs.status = 'saved' AND EXCLUDED.status <> 'saved' THEN EXCLUDED.status ELSE jobs.status END,
			applied_at  = COALESCE(jobs.applied_at, EXCLUDED.applied_at),
			updated_at  = now()
		RETURNING `+cols+`, (xmax = 0)`,
		j.UserID, j.Title, j.Company, j.Location, j.URL, j.Description, j.Salary, j.Source, j.Status, j.Notes, applied), &created)
	return out, created, err
}

// Patch updates only the non-nil fields.
type Patch struct {
	Title, Company, Location, URL, Description, Salary, Status, Notes *string
}

func (r *Repository) Patch(ctx context.Context, userID, id string, p Patch) (*Job, error) {
	j, err := scan(r.db.QueryRowContext(ctx, `
		UPDATE jobs SET
			title = COALESCE($3, title), company = COALESCE($4, company), location = COALESCE($5, location),
			url = COALESCE($6, url), description = COALESCE($7, description), salary = COALESCE($8, salary),
			status = COALESCE($9, status), notes = COALESCE($10, notes),
			applied_at = CASE WHEN $9::text IN ('applied', 'interviewing', 'offer') AND applied_at IS NULL THEN now() ELSE applied_at END,
			updated_at = now()
		WHERE user_id = $1 AND id = $2 RETURNING `+cols,
		userID, id, p.Title, p.Company, p.Location, p.URL, p.Description, p.Salary, p.Status, p.Notes))
	switch {
	case database.IsNotFound(err):
		return nil, errNotFound
	case database.IsUniqueViolation(err):
		return nil, errDuplicate
	}
	return j, err
}

func (r *Repository) SetMatch(ctx context.Context, userID, id string, score int, details json.RawMessage) error {
	_, err := r.db.ExecContext(ctx, `UPDATE jobs SET match_score = $3, match_details = $4 WHERE user_id = $1 AND id = $2`,
		userID, id, score, []byte(details))
	return err
}

func (r *Repository) Delete(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM jobs WHERE user_id = $1 AND id = $2`, userID, id)
	if database.IsInvalidInput(err) {
		return errNotFound
	} else if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func (r *Repository) StatusCounts(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, count(*) FROM jobs WHERE user_id = $1 GROUP BY status`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var s string
		var n int
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, rows.Err()
}

func (r *Repository) AppliedCounts(ctx context.Context, userID string) (total, lastWeek int, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT
		count(*) FILTER (WHERE applied_at IS NOT NULL),
		count(*) FILTER (WHERE applied_at > now() - interval '7 days')
		FROM jobs WHERE user_id = $1`, userID).Scan(&total, &lastWeek)
	return
}
