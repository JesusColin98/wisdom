package cortex

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/wisdom/pkg/observability"
	_ "modernc.org/sqlite"
)

const HNSWThreshold = 5000

// Cortex manages the semantic memory of Wisdom.
type Cortex struct {
	engine    StorageEngine
	indexPath string
	trie      *SCGTrie
}

// Open initializes or opens the Cortex database at the specified path.
func Open(path string) (*Cortex, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cortex directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cortex database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys and WAL: %w", err)
	}

	engine := NewSQLiteEngine(db, path+".rpforest")
	c := &Cortex{
		engine:    engine,
		indexPath: path + ".rpforest",
		trie:      engine.trie, // Use the engine's trie
	}
	engine.substrate = &FlatSubstrate{engine: engine}

	// Try loading existing index
	if _, err := os.Stat(c.indexPath); err == nil {
		forest := NewRPForestSubstrate(engine, 10, 768)
		if err := forest.Load(c.indexPath); err == nil {
			engine.substrate = forest
			observability.Logger.Info("Loaded RPForest index from disk", "path", c.indexPath)
		}
	}

	// Build Trie from existing nodes
	go c.WarmTrie(context.Background())

	// Immediate promotion check if existing data is large
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&count)
	if err == nil && count >= HNSWThreshold {
		if _, ok := engine.substrate.(*FlatSubstrate); ok {
			_ = c.PromoteSubstrate(context.Background())
		}
	}

	return c, nil
}

// WarmTrie builds the SCG-Mem trie from all nodes.
func (c *Cortex) WarmTrie(ctx context.Context) {
	nodes, err := c.engine.SearchNodes(ctx, "%%") // List all
	if err != nil {
		return
	}

	for _, n := range nodes {
		c.trie.Insert(n.Content, n.ID)
		c.trie.Insert(n.ID, n.ID)
	}
	observability.Logger.Info("SCG-Mem Trie Warmed")
}

// GetTrie returns the SCG-Mem trie.
func (c *Cortex) GetTrie() *SCGTrie {
	return c.trie
}

// Close closes the database connection.
func (c *Cortex) Close() error {
	return c.engine.Close()
}

// InitSchema applies the SQL schema to the database.
func (c *Cortex) InitSchema(ctx context.Context, schemaSQL string) error {
	// This is a bit leakier since we need sql.DB, but for now we'll keep it simple
	if sqlite, ok := c.engine.(*SQLiteEngine); ok {
		_, err := sqlite.db.ExecContext(ctx, schemaSQL)
		return err
	}
	return fmt.Errorf("InitSchema not supported for this engine")
}

// DB returns the underlying sql.DB connection.
func (c *Cortex) DB() *sql.DB {
	if sqlite, ok := c.engine.(*SQLiteEngine); ok {
		return sqlite.db
	}
	return nil
}

// Interaction matches the Thalamus structure but defined here for persistence.
type Interaction struct {
	Role    string
	Content string
}

// AddLog persists a session interaction to the database.
func (c *Cortex) AddLog(ctx context.Context, sessionID, role, content string) error {
	return c.engine.AddLog(ctx, sessionID, role, content)
}

// GetLogs retrieves all interactions for a specific session.
func (c *Cortex) GetLogs(ctx context.Context, sessionID string) ([]Interaction, error) {
	return c.engine.GetLogs(ctx, sessionID)
}

// ClearLogs deletes all interactions for a session.
func (c *Cortex) ClearLogs(ctx context.Context, sessionID string) error {
	return c.engine.ClearLogs(ctx, sessionID)
}

// GetInactiveSessions retrieves IDs of sessions that haven't been updated for the given duration.
func (c *Cortex) GetInactiveSessions(ctx context.Context, olderThan time.Duration) ([]string, error) {
	return c.engine.GetInactiveSessions(ctx, olderThan)
}

// PutTool stores a dynamic tool definition.
func (c *Cortex) PutTool(ctx context.Context, id, name, desc, source string) error {
	db := c.DB()
	if db == nil {
		return fmt.Errorf("no database connection available")
	}
	query := `
		INSERT INTO tools (id, name, description, source_code, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			source_code = excluded.source_code,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.ExecContext(ctx, query, id, name, desc, source)
	return err
}

// ListTools retrieves all dynamic tools from storage.
func (c *Cortex) ListTools(ctx context.Context) (map[string]string, error) {
	db := c.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection available")
	}
	rows, err := db.QueryContext(ctx, `SELECT id, source_code FROM tools`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tools := make(map[string]string)
	for rows.Next() {
		var id, source string
		if err := rows.Scan(&id, &source); err != nil {
			return nil, err
		}
		tools[id] = source
	}
	return tools, nil
}
