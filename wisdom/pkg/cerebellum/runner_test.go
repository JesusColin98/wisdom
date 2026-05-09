package cerebellum

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

type mockAsyncTool struct {
	executeFunc func(ctx context.Context, params json.RawMessage) (*Result, error)
}

func (m *mockAsyncTool) Execute(ctx context.Context, params json.RawMessage) (*Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return &Result{Success: true}, nil
}

func TestRunner_ExecuteAsync_Lifecycle(t *testing.T) {
	registry := NewRegistry()
	runner := NewRunner(registry, 2)
	ctx := context.Background()

	// Register a tool that takes some time to execute
	startSignal := make(chan struct{})
	doneSignal := make(chan struct{})
	tool := &mockAsyncTool{
		executeFunc: func(ctx context.Context, params json.RawMessage) (*Result, error) {
			startSignal <- struct{}{}
			<-doneSignal
			return &Result{Success: true, Output: "done"}, nil
		},
	}
	registry.Register(ToolDefinition{ID: "slow-tool"}, tool)

	// Execute tool
	jobID, err := runner.ExecuteAsync(ctx, "slow-tool", nil)
	if err != nil {
		t.Fatalf("ExecuteAsync failed: %v", err)
	}

	// Verify Pending or Running status
	job, _ := runner.GetJob(jobID)
	if job.Status != JobStatusPending && job.Status != JobStatusRunning {
		t.Errorf("Expected Pending or Running status, got %s", job.Status)
	}

	// Wait for tool to start
	<-startSignal
	job, _ = runner.GetJob(jobID)
	if job.Status != JobStatusRunning {
		t.Errorf("Expected Running status, got %s", job.Status)
	}

	// Finish execution
	doneSignal <- struct{}{}

	// Wait for job to update (polled for simplicity in test)
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		job, _ = runner.GetJob(jobID)
		if job.Status == JobStatusFinished {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if job.Status != JobStatusFinished {
		t.Errorf("Expected Finished status, got %s", job.Status)
	}
	if job.Result == nil || job.Result.Output != "done" {
		t.Errorf("Expected result output 'done', got %v", job.Result)
	}
	if job.FinishedAt.IsZero() {
		t.Error("Expected FinishedAt to be set")
	}
}

func TestRunner_ExecuteAsync_Error(t *testing.T) {
	registry := NewRegistry()
	runner := NewRunner(registry, 1)
	ctx := context.Background()

	tool := &mockAsyncTool{
		executeFunc: func(ctx context.Context, params json.RawMessage) (*Result, error) {
			return nil, fmt.Errorf("tool error")
		},
	}
	registry.Register(ToolDefinition{ID: "fail-tool"}, tool)

	jobID, _ := runner.ExecuteAsync(ctx, "fail-tool", nil)

	// Wait for job to fail
	deadline := time.Now().Add(1 * time.Second)
	var job *Job
	for time.Now().Before(deadline) {
		job, _ = runner.GetJob(jobID)
		if job.Status == JobStatusFailed {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if job.Status != JobStatusFailed {
		t.Errorf("Expected Failed status, got %s", job.Status)
	}
	if job.ErrorMessage != "tool error" {
		t.Errorf("Expected error message 'tool error', got '%s'", job.ErrorMessage)
	}
}

func TestRunner_ConcurrencyLimit(t *testing.T) {
	registry := NewRegistry()
	maxWorkers := 2
	runner := NewRunner(registry, maxWorkers)
	ctx := context.Background()

	var runningCount int
	var mu sync.Mutex
	startChan := make(chan struct{})
	releaseChan := make(chan struct{})

	tool := &mockAsyncTool{
		executeFunc: func(ctx context.Context, params json.RawMessage) (*Result, error) {
			mu.Lock()
			runningCount++
			mu.Unlock()
			startChan <- struct{}{}
			<-releaseChan
			mu.Lock()
			runningCount--
			mu.Unlock()
			return &Result{Success: true}, nil
		},
	}
	registry.Register(ToolDefinition{ID: "block-tool"}, tool)

	// Start 3 jobs
	for i := 0; i < 3; i++ {
		runner.ExecuteAsync(ctx, "block-tool", nil)
	}

	// Wait for 2 jobs to start
	<-startChan
	<-startChan

	// Verify only 2 are running
	mu.Lock()
	if runningCount != 2 {
		t.Errorf("Expected 2 running jobs, got %d", runningCount)
	}
	mu.Unlock()

	// Verify 3rd job is still Pending (it might be PENDING in our jobs map, even if runJob goroutine is waiting on semaphore)
	// We can't easily distinguish between "waiting for semaphore" and "not started yet" from the JobStatus alone
	// since we set it to RUNNING *after* acquiring semaphore.
	// So 3rd job should be PENDING.

	// Release one
	releaseChan <- struct{}{}

	// Wait for 3rd to start
	<-startChan

	mu.Lock()
	if runningCount != 2 {
		t.Errorf("Expected 2 running jobs after one release and one start, got %d", runningCount)
	}
	mu.Unlock()

	// Clean up
	releaseChan <- struct{}{}
	releaseChan <- struct{}{}
}
