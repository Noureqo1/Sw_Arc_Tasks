package executor

import (
	"context"
	"log"
	"time"

	"platform/functions/internal/domain/function"
	"platform/functions/internal/domain/job"

	"github.com/google/uuid"
)

type ExecRequest struct {
	JobID      uuid.UUID
	FunctionID uuid.UUID
}

type Executor struct {
	jobRepo    job.Repository
	funcRepo   function.Repository
	runner     Runner
	jobs       chan ExecRequest
	quit       chan struct{}
	numWorkers int
	timeout    time.Duration
}

func NewExecutor(
	jobRepo job.Repository,
	funcRepo function.Repository,
	runner Runner,
	numWorkers int,
) *Executor {
	return &Executor{
		jobRepo:    jobRepo,
		funcRepo:   funcRepo,
		runner:     runner,
		jobs:       make(chan ExecRequest),
		quit:       make(chan struct{}),
		numWorkers: numWorkers,
		timeout:    30 * time.Second, // 30 second timeout for jobs
	}
}

func (e *Executor) Start() {
	for i := 0; i < e.numWorkers; i++ {
		go e.workerLoop(i)
	}
}

func (e *Executor) Stop() {
	close(e.quit)
}

func (e *Executor) Enqueue(req ExecRequest) {
	e.jobs <- req
}

func (e *Executor) workerLoop(workerID int) {
	for {
		select {
		case req := <-e.jobs:
			if err := e.processRequest(req); err != nil {
				log.Printf("[worker %d] error processing request: %v\n", workerID, err)
			}
		case <-e.quit:
			return
		}
	}
}

func (e *Executor) processRequest(req ExecRequest) error {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	j, err := e.jobRepo.GetByID(ctx, req.JobID)
	if err != nil {
		return err
	}

	j.MarkRunning()
	if err := e.jobRepo.Update(ctx, j); err != nil {
		return err
	}

	fn, err := e.funcRepo.GetByID(ctx, req.FunctionID)
	if err != nil {
		j.MarkError("function not found")
		_ = e.jobRepo.Update(ctx, j)
		return err
	}

	result, runErr := e.runner.Run(ctx, fn)
	if runErr != nil {
		if runErr == context.DeadlineExceeded {
			j.MarkError("job timed out after 30 seconds")
		} else {
			j.MarkError(runErr.Error())
		}
		_ = e.jobRepo.Update(ctx, j)
		return runErr
	}

	j.MarkDone(result)
	if err := e.jobRepo.Update(ctx, j); err != nil {
		return err
	}

	return nil
}
