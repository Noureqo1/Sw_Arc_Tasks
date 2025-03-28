package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"platform/functions/internal/domain/function"
	"platform/functions/internal/domain/job"
	"platform/functions/internal/executor"
)

func SetupRoutes(
	r *gin.Engine,
	funcRepo function.Repository,
	jobRepo job.Repository,
	execSvc *executor.Executor,
) {
	h := &handler{
		funcRepo: funcRepo,
		jobRepo:  jobRepo,
		exec:     execSvc,
	}

	r.POST("/functions", h.createFunction)
	r.GET("/functions", h.listFunctions)

	r.POST("/functions/:id/execute", h.executeFunction)
	r.GET("/jobs/:id", h.getJob)
}

type handler struct {
	funcRepo function.Repository
	jobRepo  job.Repository
	exec     *executor.Executor
}

// createFunction -> POST /functions
func (h *handler) createFunction(c *gin.Context) {
	var req struct {
		Owner    string `json:"owner"`
		Code     string `json:"code"`
		Language string `json:"language"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	fmt.Print(req)

	fn := function.NewFunction(req.Owner, req.Code, req.Language)
	ctx := context.Background()
	if err := h.funcRepo.Create(ctx, fn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"function_id": fn.ID.String(), "status": "created"})
}

// listFunctions -> GET /functions
func (h *handler) listFunctions(c *gin.Context) {
	ctx := context.Background()
	funcs, err := h.funcRepo.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, funcs)
}

// executeFunction -> POST /functions/:id/execute
func (h *handler) executeFunction(c *gin.Context) {
	fnIDStr := c.Param("id")
	fnID, err := uuid.Parse(fnIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function ID"})
		return
	}

	ctx := context.Background()
	// create a new job in "queued" state
	newJob := job.NewJob(fnID)
	if err := h.jobRepo.Create(ctx, newJob); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enqueue to the Executor
	h.exec.Enqueue(executor.ExecRequest{
		JobID:      newJob.ID,
		FunctionID: fnID,
	})

	// Return the job ID so client can poll
	c.JSON(http.StatusAccepted, gin.H{
		"job_id": newJob.ID.String(),
		"status": "queued",
	})
}

// getJob -> GET /jobs/:id
func (h *handler) getJob(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	ctx := context.Background()
	j, err := h.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		if err == job.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, j)
}
