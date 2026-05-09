package thalamus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/wisdom/pkg/observability"
	lru "github.com/hashicorp/golang-lru/v2"
)

// Cache handles local-first session and context storage.
type Cache struct {
	sessions *lru.Cache[string, *Session]
}

// NewCache creates a new Thalamic cache.
func NewCache(size int) (*Cache, error) {
	c, err := lru.New[string, *Session](size)
	if err != nil {
		return nil, fmt.Errorf("failed to create lru cache: %w", err)
	}
	return &Cache{sessions: c}, nil
}

// GetSession retrieves a session from the cache.
func (c *Cache) GetSession(id string) (*Session, bool) {
	return c.sessions.Get(id)
}

// PutSession adds or updates a session in the cache.
func (c *Cache) PutSession(s *Session) {
	c.sessions.Add(s.ID, s)
}

// Warm pre-populates the cache with recent sessions from storage.
func (c *Cache) Warm(ctx context.Context, storage interface {
	GetInactiveSessions(ctx context.Context, olderThan time.Duration) ([]string, error)
}) error {
	ctx, span := observability.Tracer.Start(ctx, "Thalamus.Cache.Warm")
	defer span.End()

	observability.Logger.Info("Thalamic cache warming initiated")

	// Fetch sessions from the last 24 hours
	sessions, err := storage.GetInactiveSessions(ctx, 24*time.Hour)
	if err != nil {
		return err
	}

	for _, sid := range sessions {
		// Populate with shell sessions
		c.PutSession(&Session{
			ID:        sid,
			CreatedAt: time.Now(),
		})
	}

	observability.Logger.Info("Thalamic cache warmed", "sessions_loaded", len(sessions))
	return nil
}
