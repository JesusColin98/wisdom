package cortex

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/google/wisdom/pkg/observability"
	_ "modernc.org/sqlite"
)

const HNSWThreshold = 5000

// Cortex manages the semantic memory of Wisdom.
type Cortex struct {
	db        *sql.DB
	substrate VectorSubstrate
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

	c := &Cortex{
		db:        db,
		indexPath: path + ".rpforest",
		trie:      NewSCGTrie(),
	}
	c.substrate = &FlatSubstrate{storage: c}

	// Try loading existing index
	if _, err := os.Stat(c.indexPath); err == nil {
		forest := NewRPForestSubstrate(c, 10, 768)
		if err := forest.Load(c.indexPath); err == nil {
			c.substrate = forest
			observability.Logger.Info("Loaded RPForest index from disk", "path", c.indexPath)
		}
	}

	// Build Trie from existing nodes
	go c.WarmTrie(context.Background())

	// Immediate promotion check if existing data is large
	var count int
	err = c.db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&count)
	if err == nil && count >= HNSWThreshold {
		if _, ok := c.substrate.(*FlatSubstrate); ok {
			_ = c.PromoteSubstrate(context.Background())
		}
	}

	return c, nil
}

// WarmTrie builds the SCG-Mem trie from all nodes.
func (c *Cortex) WarmTrie(ctx context.Context) {
	query := `SELECT id, content FROM nodes`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, content string
		if err := rows.Scan(&id, &content); err == nil {
			c.trie.Insert(content, id)
			c.trie.Insert(id, id) // Also index the ID itself
		}
	}
	observability.Logger.Info("SCG-Mem Trie Warmed")
}

// GetTrie returns the SCG-Mem trie.
func (c *Cortex) GetTrie() *SCGTrie {
	return c.trie
}

// Close closes the database connection.
func (c *Cortex) Close() error {
	return c.db.Close()
}

// InitSchema applies the SQL schema to the database.
func (c *Cortex) InitSchema(ctx context.Context, schemaSQL string) error {
	_, err := c.db.ExecContext(ctx, schemaSQL)
	return err
}

// DB returns the underlying sql.DB connection.
func (c *Cortex) DB() *sql.DB {
	return c.db
}

// Interaction matches the Thalamus structure but defined here for persistence.
type Interaction struct {
	Role    string
	Content string
}

// AddLog persists a session interaction to the database.
func (c *Cortex) AddLog(ctx context.Context, sessionID, role, content string) error {
	query := `INSERT INTO session_logs (session_id, role, content) VALUES (?, ?, ?)`
	_, err := c.db.ExecContext(ctx, query, sessionID, role, content)
	return err
}

// GetLogs retrieves all interactions for a specific session.
func (c *Cortex) GetLogs(ctx context.Context, sessionID string) ([]Interaction, error) {
	query := `SELECT role, content FROM session_logs WHERE session_id = ? ORDER BY log_id ASC`
	rows, err := c.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []Interaction
	for rows.Next() {
		var i Interaction
		if err := rows.Scan(&i.Role, &i.Content); err != nil {
			return nil, err
		}
		logs = append(logs, i)
	}
	return logs, nil
}

// ClearLogs deletes all interactions for a session.
func (c *Cortex) ClearLogs(ctx context.Context, sessionID string) error {
	query := `DELETE FROM session_logs WHERE session_id = ?`
	_, err := c.db.ExecContext(ctx, query, sessionID)
	return err
}

// GetInactiveSessions retrieves IDs of sessions that haven't been updated for the given duration.
func (c *Cortex) GetInactiveSessions(ctx context.Context, olderThan time.Duration) ([]string, error) {
	query := `
		SELECT session_id
		FROM session_logs
		GROUP BY session_id
		HAVING MAX(created_at) < datetime('now', ?)
	`
	// olderThan in seconds (e.g., "-24 hours")
	offset := fmt.Sprintf("-%d seconds", int(olderThan.Seconds()))
	rows, err := c.db.QueryContext(ctx, query, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		sessions = append(sessions, id)
	}
	return sessions, nil
}

// floatsToBytes converts []float32 to a byte slice for storage.
func floatsToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// bytesToFloats converts a byte slice back to []float32.
func bytesToFloats(b []byte) []float32 {
	floats := make([]float32, len(b)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return floats
}

// PutTool stores a dynamic tool definition.
func (c *Cortex) PutTool(ctx context.Context, id, name, desc, source string) error {
	query := `
		INSERT INTO tools (id, name, description, source_code, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			source_code = excluded.source_code,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := c.db.ExecContext(ctx, query, id, name, desc, source)
	return err
}

// ListTools retrieves all dynamic tools from storage.
func (c *Cortex) ListTools(ctx context.Context) (map[string]string, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT id, source_code FROM tools`)
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
