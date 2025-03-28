package db

import (
	"fmt"
	"github.com/jmoiron/sqlx"
)

func RunMigrations(db *sqlx.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS functions (
			id UUID PRIMARY KEY,
			owner TEXT NOT NULL,
			code TEXT NOT NULL,
			language TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			id UUID PRIMARY KEY,
			function_id UUID NOT NULL,
			status TEXT NOT NULL,
			result TEXT,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration query failed: %w", err)
		}
	}
	return nil
}
