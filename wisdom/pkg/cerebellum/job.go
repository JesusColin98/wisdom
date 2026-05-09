package cerebellum

import (
	"time"
)

// JobStatus represents the current state of an asynchronous job.
type JobStatus string

const (
	// JobStatusPending indicates the job has been created but not yet started.
	JobStatusPending JobStatus = "PENDING"
	// JobStatusRunning indicates the job is currently being executed.
	JobStatusRunning JobStatus = "RUNNING"
	// JobStatusFinished indicates the job completed successfully.
	JobStatusFinished JobStatus = "FINISHED"
	// JobStatusFailed indicates the job failed during execution.
	JobStatusFailed JobStatus = "FAILED"
)

// Job represents an asynchronous tool execution request and its state.
type Job struct {
	// ID is the unique identifier for this job.
	ID string `json:"id"`
	// Status is the current lifecycle state of the job.
	Status JobStatus `json:"status"`
	// Result contains the outcome of the tool execution if successful.
	Result *Result `json:"result,omitempty"`
	// Error contains the error message if the job failed.
	Error error `json:"-"`
	// ErrorMessage is a string representation of Error for JSON serialization.
	ErrorMessage string `json:"error,omitempty"`
	// CreatedAt is the timestamp when the job was first submitted.
	CreatedAt time.Time `json:"created_at"`
	// FinishedAt is the timestamp when the job reached a terminal state (Finished or Failed).
	FinishedAt time.Time `json:"finished_at,omitempty"`
}
