package thalamus

import (
	"fmt"

	"github.com/google/wisdom/pkg/errors"
)

// Gate defines the admission control for tool calls.
type Gate struct {
	validator *Validator
	cache     *Cache
}

// NewGate creates a new Thalamic Gate.
func NewGate(v *Validator, c *Cache) *Gate {
	return &Gate{
		validator: v,
		cache:     c,
	}
}

// Admit checks if a tool call is permitted and valid.
func (g *Gate) Admit(sessionID, method, payload string) error {
	// 1. Session check
	session, ok := g.cache.GetSession(sessionID)
	if !ok {
		return errors.New(errors.CodeUnauthorized, fmt.Sprintf("session %s not found or expired", sessionID))
	}

	// 2. Reactive Gating: Check if the method is explicitly blocked for this session
	if session.Flags["block_"+method] == "true" {
		return errors.New(errors.CodeUnauthorized, fmt.Sprintf("method %s is restricted for this session", method))
	}

	// 3. Schema Validation
	if err := g.validator.Validate(method, payload); err != nil {
		return errors.Wrap(errors.CodeInvalidParams, "Parameter validation failed at the Thalamic Gate", err)
	}

	return nil
}
