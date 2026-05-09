package cerebellum

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/wisdom/pkg/errors"
	"github.com/google/wisdom/pkg/observability"
)

// ErrNotFound is returned when a requested tool does not exist in the registry.
var ErrNotFound = errors.New(errors.CodeNotFound, "tool not found")

// ErrCircuitOpen is returned when a tool's circuit breaker is in the open state.
var ErrCircuitOpen = errors.New(errors.CodeUnavailable, "tool circuit breaker is open")

// toolEntry combines a tool's definition and its implementation.
type toolEntry struct {
	definition     ToolDefinition
	implementation Tool
}

// Registry provides a thread-safe central registry for tool management.
type Registry struct {
	mu       sync.RWMutex
	tools    map[string]toolEntry
	circuits map[string]*CircuitBreaker
}

// NewRegistry creates a new instance of the tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:    make(map[string]toolEntry),
		circuits: make(map[string]*CircuitBreaker),
	}
}

// Register adds a new tool to the registry. It returns an error if a tool with
// the same ID is already registered.
func (r *Registry) Register(def ToolDefinition, impl Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tools[def.ID]; ok {
		return errors.New(errors.CodeInternal, fmt.Sprintf("tool with ID %s already registered", def.ID))
	}

	r.tools[def.ID] = toolEntry{
		definition:     def,
		implementation: impl,
	}
	// Initialize default circuit breaker: threshold 5, timeout 30s
	r.circuits[def.ID] = NewCircuitBreaker(5, 30*time.Second)
	return nil
}

// Get retrieves a tool's definition and implementation by its ID.
// It instruments the lookup with OTel tracing and checks the circuit breaker.
func (r *Registry) Get(ctx context.Context, id string) (ToolDefinition, Tool, error) {
	ctx, span := observability.Tracer.Start(ctx, "cerebellum.Registry.Get")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.tools[id]
	if !ok {
		return ToolDefinition{}, nil, ErrNotFound
	}

	circuit, ok := r.circuits[id]
	if ok && !circuit.Allow() {
		return ToolDefinition{}, nil, ErrCircuitOpen
	}

	return entry.definition, entry.implementation, nil
}

// ReportResult updates the circuit breaker state based on the tool execution result.
func (r *Registry) ReportResult(id string, success bool) {
	r.mu.RLock()
	circuit, ok := r.circuits[id]
	r.mu.RUnlock()

	if ok {
		if success {
			circuit.RecordSuccess()
		} else {
			circuit.RecordFailure()
		}
	}
}

// List returns a list of all registered tool definitions.
func (r *Registry) List() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]ToolDefinition, 0, len(r.tools))
	for _, entry := range r.tools {
		list = append(list, entry.definition)
	}
	return list
}

// LoadDynamicTools restores tools from the Cortex.
func (r *Registry) LoadDynamicTools(ctx context.Context, storage interface {
	ListTools(context.Context) (map[string]string, error)
}) error {
	dynamicTools, err := storage.ListTools(ctx)
	if err != nil {
		return err
	}

	for id, source := range dynamicTools {
		// Mock definition since we only stored source in ListTools for now
		def := ToolDefinition{ID: id, Name: id, Description: "Restored dynamic tool"}
		
		dt, err := Synthesize(def, source)
		if err != nil {
			observability.Logger.Error("Failed to restore dynamic tool", "id", id, "error", err)
			continue
		}

		if err := r.Register(def, dt); err != nil {
			observability.Logger.Error("Failed to register restored tool", "id", id, "error", err)
			continue
		}
		
		observability.Logger.Info("Restored dynamic tool", "id", id)
	}
	return nil
}
