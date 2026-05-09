package cerebellum

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestRegistry_CircuitBreakerIntegration(t *testing.T) {
	registry := NewRegistry()
	def := ToolDefinition{ID: "test-tool"}
	tool := &mockAsyncTool{
		executeFunc: func(ctx context.Context, params json.RawMessage) (*Result, error) {
			return &Result{Success: false}, nil
		},
	}
	registry.Register(def, tool)

	ctx := context.Background()

	// Threshold is 5 by default. Record 5 failures.
	for i := 0; i < 5; i++ {
		_, _, err := registry.Get(ctx, "test-tool")
		if err != nil {
			t.Fatalf("expected Get to succeed on attempt %d, got %v", i+1, err)
		}
		registry.ReportResult("test-tool", false)
	}

	// Next Get should fail with ErrCircuitOpen
	_, _, err := registry.Get(ctx, "test-tool")
	if err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	// Report success should close it
	registry.ReportResult("test-tool", true)
	_, _, err = registry.Get(ctx, "test-tool")
	if err != nil {
		t.Fatalf("expected Get to succeed after success report, got %v", err)
	}
}

func TestRunner_CircuitBreakerIntegration(t *testing.T) {
	registry := NewRegistry()
	runner := NewRunner(registry, 1)

	// Tool that always fails
	tool := &mockAsyncTool{
		executeFunc: func(ctx context.Context, params json.RawMessage) (*Result, error) {
			return &Result{Success: false}, nil
		},
	}
	// threshold 2 for faster test
	def := ToolDefinition{ID: "fail-tool"}
	registry.mu.Lock()
	registry.tools[def.ID] = toolEntry{definition: def, implementation: tool}
	registry.circuits[def.ID] = NewCircuitBreaker(2, 10*time.Second)
	registry.mu.Unlock()

	ctx := context.Background()

	// 1st failure
	jobID, _ := runner.ExecuteAsync(ctx, "fail-tool", nil)
	waitForJob(runner, jobID, JobStatusFailed)

	// Circuit should still be closed
	if !registry.circuits["fail-tool"].Allow() {
		t.Fatal("expected circuit to be closed after 1 failure")
	}

	// 2nd failure
	jobID, _ = runner.ExecuteAsync(ctx, "fail-tool", nil)
	waitForJob(runner, jobID, JobStatusFailed)

	// Circuit should be open
	if registry.circuits["fail-tool"].Allow() {
		t.Fatal("expected circuit to be open after 2 failures")
	}

	// ExecuteAsync should now return ErrCircuitOpen immediately because Get() calls Allow()
	_, err := runner.ExecuteAsync(ctx, "fail-tool", nil)
	if err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func waitForJob(runner *Runner, jobID string, target JobStatus) {
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		job, _ := runner.GetJob(jobID)
		if job.Status == target {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
