package job

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("job not found")

type Repository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, jobID uuid.UUID) (*Job, error)
	Update(ctx context.Context, job *Job) error
}
