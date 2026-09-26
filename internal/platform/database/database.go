// Package database opens the Postgres pool and classifies driver errors.
// The schema lives in the applypilot-migrations repo (Soda).
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

func Open(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	var pingErr error
	for i := 0; i < 30; i++ {
		if pingErr = db.PingContext(ctx); pingErr == nil {
			return db, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("database not reachable: %w", pingErr)
}

// CheckSchema fails fast with a helpful message when migrations haven't run.
func CheckSchema(ctx context.Context, db *sql.DB) error {
	var ok bool
	err := db.QueryRowContext(ctx, `SELECT to_regclass('public.ai_generations') IS NOT NULL`).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("database schema missing: run `make up` in applypilot-migrations first")
	}
	return nil
}

func pqCode(err error) string {
	var pe *pq.Error
	if errors.As(err, &pe) {
		return string(pe.Code)
	}
	return ""
}

// IsUniqueViolation reports a unique-constraint conflict (23505).
func IsUniqueViolation(err error) bool { return pqCode(err) == "23505" }

// IsInvalidInput reports malformed input such as a non-UUID id (22P02).
func IsInvalidInput(err error) bool { return pqCode(err) == "22P02" }

// IsNotFound reports no rows, or an id that can't exist because it's malformed.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) || IsInvalidInput(err) }

// Tx runs fn in a transaction, rolling back on error.
func Tx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
