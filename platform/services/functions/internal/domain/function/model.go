package function

import (
	"time"

	"github.com/google/uuid"
)

type Function struct {
	ID        uuid.UUID `db:"id"`
	Owner     string    `db:"owner"`
	Code      string    `db:"code"`
	Language  string    `db:"language"`
	CreatedAt time.Time `db:"created_at"`
}

func NewFunction(owner, code, language string) *Function {
	return &Function{
		ID:        uuid.New(),
		Owner:     owner,
		Code:      code,
		Language:  language,
		CreatedAt: time.Now(),
	}
}
