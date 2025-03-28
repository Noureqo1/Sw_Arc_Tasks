package function

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

func (r *postgresRepo) Create(ctx context.Context, fn *Function) error {
	const query = `
	INSERT INTO functions (id, owner, code, language, created_at)
	VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		fn.ID, fn.Owner, fn.Code, fn.Language, fn.CreatedAt)
	return err
}

func (r *postgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*Function, error) {
	const query = `
	SELECT id, owner, code, language, created_at
	  FROM functions
	 WHERE id = $1
	`
	var row Function
	err := r.db.GetContext(ctx, &row, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (r *postgresRepo) List(ctx context.Context) ([]Function, error) {
	const query = `
	SELECT id, owner, code, language, created_at
	  FROM functions
	 ORDER BY created_at DESC
	`
	var funcs []Function
	if err := r.db.SelectContext(ctx, &funcs, query); err != nil {
		return nil, err
	}
	return funcs, nil
}
