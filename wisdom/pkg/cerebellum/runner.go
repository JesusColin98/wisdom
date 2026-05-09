package cerebellum

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/google/wisdom/pkg/errors"
	"github.com/google/wisdom/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Runner manages asynchronous tool execution with concurrency control.
type Runner struct {
	registry   *Registry
	maxWorkers int
	sem        chan struct{}
	jobs       map[string]*Job
	mu         sync.RWMutex
}

// NewRunner creates a new Runner with the specified tool registry and worker limit.
func NewRunner(registry *Registry, maxWorkers int) *Runner {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	return &Runner{
		registry:   registry,
		maxWorkers: maxWorkers,
		sem:        make(chan struct{}, maxWorkers),
		jobs:       make(map[string]*Job),
	}
}

// ExecuteAsync schedules a tool for asynchronous execution.
// It returns a unique Job ID immediately.
func (r *Runner) ExecuteAsync(ctx context.Context, toolID string, params json.RawMessage) (string, error) {
	ctx, span := observability.Tracer.Start(ctx, "cerebellum.Runner.ExecuteAsync",
		trace.WithAttributes(attribute.String("tool_id", toolID)))
	defer span.End()

	_, tool, err := r.registry.Get(ctx, toolID)
	if err != nil {
		return "", err
	}

	jobID := uuid.New().String()
	job := &Job{
		ID:        jobID,
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
	}

	r.mu.Lock()
	r.jobs[jobID] = job
	r.mu.Unlock()

	// Start execution in a background goroutine
	go r.runJob(context.Background(), jobID, toolID, tool, params)

	return jobID, nil
}

// GetJob retrieves a job's current state by its ID.
func (r *Runner) GetJob(id string) (*Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[id]
	if !ok {
		return nil, errors.New(errors.CodeNotFound, "job not found")
	}

	return job, nil
}

// runJob handles the execution lifecycle of a single job.
func (r *Runner) runJob(ctx context.Context, jobID string, toolID string, tool Tool, params json.RawMessage) {
	// Wait for an available worker slot
	r.sem <- struct{}{}
	defer func() { <-r.sem }()

	r.updateJobStatus(jobID, JobStatusRunning, nil, nil)

	// Execute the tool
	result, err := tool.Execute(ctx, params)

	if err != nil {
		r.registry.ReportResult(toolID, false)
		r.updateJobStatus(jobID, JobStatusFailed, nil, err)
	} else {
		r.registry.ReportResult(toolID, result.Success)
		if result.Success {
			r.updateJobStatus(jobID, JobStatusFinished, result, nil)
		} else {
			r.updateJobStatus(jobID, JobStatusFailed, result, errors.New(errors.CodeInternal, "tool execution failed"))
		}
	}
}

// updateJobStatus updates the state of a job in a thread-safe manner.
func (r *Runner) updateJobStatus(id string, status JobStatus, result *Result, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.jobs[id]
	if !ok {
		return
	}

	job.Status = status
	if result != nil {
		job.Result = result
	}
	if err != nil {
		job.Error = err
		job.ErrorMessage = err.Error()
	}

	if status == JobStatusFinished || status == JobStatusFailed {
		job.FinishedAt = time.Now()
	}
}
