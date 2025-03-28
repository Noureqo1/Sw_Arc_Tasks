package function

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, fn *Function) error
	GetByID(ctx context.Context, id uuid.UUID) (*Function, error)
	List(ctx context.Context) ([]Function, error)
}

var ErrNotFound = errors.New("function not found")
