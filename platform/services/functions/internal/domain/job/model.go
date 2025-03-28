package job

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQueued  Status = "queued"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

type Job struct {
	ID         uuid.UUID `db:"id"`
	FunctionID uuid.UUID `db:"function_id"`
	Status     Status    `db:"status"`
	Result     string    `db:"result"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func NewJob(functionID uuid.UUID) *Job {
	now := time.Now()
	return &Job{
		ID:         uuid.New(),
		FunctionID: functionID,
		Status:     StatusQueued,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (j *Job) MarkRunning() {
	j.Status = StatusRunning
	j.UpdatedAt = time.Now()
}

func (j *Job) MarkDone(result string) {
	j.Status = StatusDone
	j.Result = result
	j.UpdatedAt = time.Now()
}

func (j *Job) MarkError(errMsg string) {
	j.Status = StatusError
	j.Result = errMsg
	j.UpdatedAt = time.Now()
}
