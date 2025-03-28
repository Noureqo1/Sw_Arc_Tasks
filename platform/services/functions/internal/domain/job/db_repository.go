package job

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type postgresRepo struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) Repository {
	return &postgresRepo{db: db}
}

func (r *postgresRepo) Create(ctx context.Context, j *Job) error {
	const query = `
	  INSERT INTO jobs (id, function_id, status, result, created_at, updated_at)
	  VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		j.ID, j.FunctionID, j.Status, j.Result, j.CreatedAt, j.UpdatedAt)
	return err
}

func (r *postgresRepo) GetByID(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	const query = `
	  SELECT id, function_id, status, result, created_at, updated_at
	    FROM jobs
	   WHERE id = $1
	`
	var row Job
	err := r.db.GetContext(ctx, &row, query, jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *postgresRepo) Update(ctx context.Context, j *Job) error {
	const query = `
	  UPDATE jobs
	     SET status    = $1,
	         result    = $2,
	         updated_at= $3
	   WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, j.Status, j.Result, j.UpdatedAt, j.ID)
	return err
}
