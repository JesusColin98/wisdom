package thalamus

import (
	"time"
)

// Session represents an active cognitive session.
type Session struct {
	ID        string
	User      string
	Flags     map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Context represents the aggregated context for a session.
type Context struct {
	Session    *Session
	Wisdom     []string
	Budget     int
	Strictness Strictness
}

// NewSession creates a new session with defaults.
func NewSession(id, user string) *Session {
	return &Session{
		ID:        id,
		User:      user,
		Flags:     make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
