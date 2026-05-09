package cortex

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/wisdom/pkg/observability"
	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"
)

// ... existing code ...

const HNSWThreshold = 5000

// PromoteSubstrate upgrades the vector search substrate to Tier 2 (RPForest)
func (c *Cortex) PromoteSubstrate(ctx context.Context) error {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.PromoteSubstrate")
	defer span.End()

	if _, ok := c.substrate.(*RPForestSubstrate); ok {
		return nil // Already promoted
	}

	observability.Logger.Info("Promoting substrate to RPForest Tier")

	// 1. Fetch all existing vectors
	query := `SELECT node_id, embedding FROM vectors`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 2. Build Forest
	// Assuming 768 dimensions (typical for many models) or detect from first row
	forest := NewRPForestSubstrate(c, 10, 768) 
	
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			continue
		}
		vec := bytesToFloats(blob)
		if len(vec) > 0 {
			forest.Dim = len(vec) // Adjust dimension if needed
			_ = forest.Add(ctx, id, vec)
		}
	}

	// 3. Swap Substrate
	c.substrate = forest
	return forest.Save(c.indexPath)
}

func (c *Cortex) scoreSpecificNodes(ctx context.Context, queryEmbedding []float32, ids []string, topK int) ([]ScoredNode, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build IN clause
	placeholders := ""
	args := make([]any, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.created_at, n.updated_at, v.embedding
		FROM nodes n
		JOIN vectors v ON n.id = v.node_id
		WHERE n.id IN (%s)
	`, placeholders)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ScoredNode
	for rows.Next() {
		var sn ScoredNode
		var metadataRaw []byte
		var embeddingRaw []byte
		err := rows.Scan(
			&sn.ID, &sn.Content, &sn.EntityClass, &sn.Author, &sn.SourceType,
			&sn.SourceRef, &sn.NamespaceID, &metadataRaw, &sn.ConfidenceScore,
			&sn.CreatedAt, &sn.UpdatedAt, &embeddingRaw,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataRaw, &sn.Metadata); err != nil {
			return nil, err
		}

		nodeEmbedding := bytesToFloats(embeddingRaw)
		sn.Score = cosineSimilarity(queryEmbedding, nodeEmbedding)
		results = append(results, sn)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// PutVector stores a semantic embedding for a node.
func (c *Cortex) PutVector(ctx context.Context, nodeID string, embedding []float32, modelVersion string) error {
	blob := floatsToBytes(embedding)
	query := `
		INSERT INTO vectors (node_id, embedding, model_version, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(node_id) DO UPDATE SET
			embedding = excluded.embedding,
			model_version = excluded.model_version,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := c.db.ExecContext(ctx, query, nodeID, blob, modelVersion)
	if err != nil {
		return err
	}

	// Update Substrate
	if c.substrate != nil {
		_ = c.substrate.Add(ctx, nodeID, embedding)
		if forest, ok := c.substrate.(*RPForestSubstrate); ok {
			// Save periodically? For now, save every time or just let it be in-mem and save on promotion/exit
			// Actually, let's save every 100 additions to keep it low cost
			var count int
			_ = c.db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&count)
			if count%100 == 0 {
				_ = forest.Save(c.indexPath)
			}
		}
	}

	// Async promotion check
	go func() {
		var count int
		_ = c.db.QueryRow("SELECT COUNT(*) FROM vectors").Scan(&count)
		if count >= HNSWThreshold {
			if _, ok := c.substrate.(*FlatSubstrate); ok {
				_ = c.PromoteSubstrate(context.Background())
			}
		}
	}()

	return nil
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

// Interaction matches the Thalamus structure but defined here for persistence.
type Interaction struct {
	Role    string
	Content string
}

// GetVector retrieves the semantic embedding for a node.
func (c *Cortex) GetVector(ctx context.Context, nodeID string) ([]float32, string, error) {
	query := `SELECT embedding, model_version FROM vectors WHERE node_id = ?`
	row := c.db.QueryRowContext(ctx, query, nodeID)

	var blob []byte
	var modelVersion string
	if err := row.Scan(&blob, &modelVersion); err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nil
		}
		return nil, "", err
	}

	return bytesToFloats(blob), modelVersion, nil
}

// UpdateConfidence adjusts the truth-score of a node based on feedback.
func (c *Cortex) UpdateConfidence(ctx context.Context, nodeID string, delta float64) error {
	query := `UPDATE nodes SET confidence_score = MIN(1.0, MAX(0.0, confidence_score + ?)), updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := c.db.ExecContext(ctx, query, delta, nodeID)
	return err
}

// ResolvePointer resolves a semantic alias or ID to a specific node ID.
// It prioritizes SYNONYM_OF links to find canonical entities.
func (c *Cortex) ResolvePointer(ctx context.Context, pointer string) (string, error) {
	// 1. Synonym check (Highest priority for aliases)
	query := `
		SELECT target_id 
		FROM links 
		WHERE source_id = ? AND relation_type = 'SYNONYM_OF'
		ORDER BY weight DESC LIMIT 1
	`
	var id string
	err := c.db.QueryRowContext(ctx, query, pointer).Scan(&id)
	if err == nil {
		return id, nil
	}

	// 2. Direct ID check
	err = c.db.QueryRowContext(ctx, `SELECT id FROM nodes WHERE id = ?`, pointer).Scan(&id)
	if err == nil {
		return id, nil
	}

	return "", fmt.Errorf("could not resolve pointer: %s", pointer)
}

// FindSimilar finds the most semantically similar node to the given embedding.
// It returns the node and its similarity score if it exceeds the provided threshold.
func (c *Cortex) FindSimilar(ctx context.Context, embedding []float32, threshold float64) (*ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.FindSimilar")
	defer span.End()

	// For now, use the flat VectorSearch logic. 
	// As we scale, this will automatically use the HNSW substrate.
	results, err := c.VectorSearch(ctx, embedding, 1)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 && results[0].Score >= threshold {
		return &results[0], nil
	}

	return nil, nil
}

// StrengthenSynapse increases the confidence score of an existing node.
func (c *Cortex) StrengthenSynapse(ctx context.Context, nodeID string) error {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.StrengthenSynapse")
	defer span.End()

	query := `
		UPDATE nodes 
		SET confidence_score = MIN(1.0, confidence_score + 0.05),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := c.db.ExecContext(ctx, query, nodeID)
	return err
}

// VectorSearch performs a semantic similarity search across all nodes.
func (c *Cortex) VectorSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.VectorSearch")
	defer span.End()

	// If no embedding provided, fallback to standard ranking
	if queryEmbedding == nil {
		return nil, nil
	}

	if c.substrate != nil {
		return c.substrate.Search(ctx, queryEmbedding, topK)
	}

	return c.linearVectorSearch(ctx, queryEmbedding, topK)
}

func (c *Cortex) linearVectorSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	query := `
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.created_at, n.updated_at, v.embedding
		FROM nodes n
		JOIN vectors v ON n.id = v.node_id
	`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ScoredNode
	for rows.Next() {
		var sn ScoredNode
		var metadataRaw []byte
		var embeddingRaw []byte
		err := rows.Scan(
			&sn.ID, &sn.Content, &sn.EntityClass, &sn.Author, &sn.SourceType,
			&sn.SourceRef, &sn.NamespaceID, &metadataRaw, &sn.ConfidenceScore,
			&sn.CreatedAt, &sn.UpdatedAt, &embeddingRaw,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataRaw, &sn.Metadata); err != nil {
			return nil, err
		}

		nodeEmbedding := bytesToFloats(embeddingRaw)
		sn.Score = cosineSimilarity(queryEmbedding, nodeEmbedding)
		results = append(results, sn)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

// HybridSearch combines Keyword and Vector search using Reciprocal Rank Fusion (RRF).
func (c *Cortex) HybridSearch(ctx context.Context, keywordQuery string, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.HybridSearch")
	defer span.End()

	// 1. Concurrent Search
	var keywordResults []Node
	var vectorResults []ScoredNode
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		keywordResults, err = c.SearchNodes(gCtx, keywordQuery)
		return err
	})

	g.Go(func() error {
		var err error
		vectorResults, err = c.VectorSearch(gCtx, queryEmbedding, topK*2)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// 2. Reciprocal Rank Fusion (RRF)
	// Score(d) = sum_{r in R} 1 / (k + rank(d, r))
	// where k is a constant (usually 60)
	const k = 60.0
	fusedScores := make(map[string]float64)
	nodesMap := make(map[string]Node)

	for rank, node := range keywordResults {
		fusedScores[node.ID] += 1.0 / (k + float64(rank+1))
		nodesMap[node.ID] = node
	}

	for rank, sn := range vectorResults {
		fusedScores[sn.ID] += 1.0 / (k + float64(rank+1))
		nodesMap[sn.ID] = sn.Node
	}

	// 3. Collect and Sort
	var finalResults []ScoredNode
	for id, score := range fusedScores {
		finalResults = append(finalResults, ScoredNode{
			Node:  nodesMap[id],
			Score: score,
		})
	}

	sort.Slice(finalResults, func(i, j int) bool {
		return finalResults[i].Score > finalResults[j].Score
	})

	if len(finalResults) > topK {
		finalResults = finalResults[:topK]
	}

	return finalResults, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// GetHistory retrieves the version history for a given node.
func (c *Cortex) GetHistory(ctx context.Context, nodeID string) ([]NodeHistory, error) {
	query := `
		SELECT node_id, content, metadata, version_timestamp
		FROM node_history
		WHERE node_id = ?
		ORDER BY history_id DESC
	`
	rows, err := c.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []NodeHistory
	for rows.Next() {
		var h NodeHistory
		var metadataRaw []byte
		if err := rows.Scan(&h.NodeID, &h.Content, &metadataRaw, &h.VersionTimestamp); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadataRaw, &h.Metadata); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
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

// Cortex manages the semantic memory of Wisdom.
type Cortex struct {
	db        *sql.DB
	substrate VectorSubstrate
	indexPath string
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

// DB returns the underlying sql.DB connection.
func (c *Cortex) DB() *sql.DB {
	return c.db
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
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	c := &Cortex{
		db:        db,
		indexPath: path + ".rpforest",
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

// Close closes the database connection.
func (c *Cortex) Close() error {
	return c.db.Close()
}

// InitSchema applies the SQL schema to the database.
func (c *Cortex) InitSchema(ctx context.Context, schemaSQL string) error {
	_, err := c.db.ExecContext(ctx, schemaSQL)
	return err
}

// PutNode inserts or updates a wisdom node.
func (c *Cortex) PutNode(ctx context.Context, node *Node) error {
	metadataJSON, err := json.Marshal(node.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO nodes (id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, superseded_by_id, valid_from, valid_until, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			entity_class = excluded.entity_class,
			author = excluded.author,
			source_type = excluded.source_type,
			source_ref = excluded.source_ref,
			metadata = excluded.metadata,
			confidence_score = excluded.confidence_score,
			superseded_by_id = excluded.superseded_by_id,
			valid_from = excluded.valid_from,
			valid_until = excluded.valid_until,
			updated_at = CURRENT_TIMESTAMP
	`
	var supersededBy sql.NullString
	if node.SupersededByID != "" {
		supersededBy = sql.NullString{String: node.SupersededByID, Valid: true}
	}

	_, err = c.db.ExecContext(ctx, query,
		node.ID, node.Content, node.EntityClass, node.Author, node.SourceType,
		node.SourceRef, node.NamespaceID, metadataJSON, node.ConfidenceScore,
		supersededBy, node.ValidFrom, node.ValidUntil,
	)
	if err != nil {
		observability.Logger.Error("PutNode failed", "error", err, "node_id", node.ID, "namespace_id", node.NamespaceID, "superseded_by_id", node.SupersededByID)
	}
	return err
}

// GetNode retrieves a wisdom node by ID.
func (c *Cortex) GetNode(ctx context.Context, id string) (*Node, error) {
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, superseded_by_id, valid_from, valid_until, created_at, updated_at FROM nodes WHERE id = ?`
	row := c.db.QueryRowContext(ctx, query, id)

	var node Node
	var metadataRaw []byte
	var supersededByID, validFrom, validUntil sql.NullString

	err := row.Scan(
		&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
		&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
		&supersededByID, &validFrom, &validUntil,
		&node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if supersededByID.Valid {
		node.SupersededByID = supersededByID.String
	}
	if validFrom.Valid {
		node.ValidFrom, _ = time.Parse(time.RFC3339, validFrom.String)
	}
	if validUntil.Valid {
		node.ValidUntil, _ = time.Parse(time.RFC3339, validUntil.String)
	}

	if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &node, nil
}

// CreateNamespace ensures a namespace exists.
func (c *Cortex) CreateNamespace(ctx context.Context, ns *Namespace) error {
	query := `INSERT OR IGNORE INTO namespaces (id, name, description) VALUES (?, ?, ?)`
	_, err := c.db.ExecContext(ctx, query, ns.ID, ns.Name, ns.Description)
	return err
}

// LinkNodes creates a directed relationship between two nodes.
func (c *Cortex) LinkNodes(ctx context.Context, link *Link) error {
	query := `
		INSERT INTO links (source_id, target_id, relation_type, weight)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(source_id, target_id, relation_type) DO UPDATE SET
			weight = excluded.weight
	`
	_, err := c.db.ExecContext(ctx, query, link.SourceID, link.TargetID, link.RelationType, link.Weight)
	return err
}

// GetNeighbors retrieves all nodes connected to a given node.
func (c *Cortex) GetNeighbors(ctx context.Context, nodeID string) ([]*Node, error) {
	query := `
		SELECT n.id, n.content, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.created_at, n.updated_at
		FROM nodes n
		JOIN links l ON n.id = l.target_id
		WHERE l.source_id = ?
	`
	rows, err := c.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		var node Node
		var metadataRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

// Edge represents an edge in the graph for propagation.
type Edge struct {
	SourceID string
	TargetID string
	Weight   float64
}

// GetEdges retrieves edges starting from a set of nodes.
func (c *Cortex) GetEdges(ctx context.Context, sourceIDs []string) ([]Edge, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}

	// Simple IN clause construction (assuming no SQL injection for UUIDs/IDs)
	// In production, use a library or properly escaped place holders.
	placeholders := ""
	args := make([]any, len(sourceIDs))
	for i, id := range sourceIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	query := fmt.Sprintf(`SELECT source_id, target_id, weight FROM links WHERE source_id IN (%s)`, placeholders)
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.SourceID, &e.TargetID, &e.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, nil
}

// ListNodes retrieves all nodes in a specific namespace.
func (c *Cortex) ListNodes(ctx context.Context, namespaceID string) ([]Node, error) {
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, created_at, updated_at FROM nodes WHERE namespace_id = ?`
	rows, err := c.db.QueryContext(ctx, query, namespaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		var metadataRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for node %s: %w", node.ID, err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// ListEdges retrieves all relationships (links) in the Cortex.
func (c *Cortex) ListEdges(ctx context.Context) ([]Link, error) {
	query := `SELECT source_id, target_id, relation_type, weight, created_at FROM links`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		err := rows.Scan(&link.SourceID, &link.TargetID, &link.RelationType, &link.Weight, &link.CreatedAt)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

// SearchNodes performs a keyword search on node content, ID, and entity class.
func (c *Cortex) SearchNodes(ctx context.Context, query string) ([]Node, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cortex.SearchNodes")
	defer span.End()

	sqlQuery := `
		SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, created_at, updated_at
		FROM nodes
		WHERE content LIKE ? OR id LIKE ? OR entity_class LIKE ?
	`
	likeQuery := "%" + query + "%"
	rows, err := c.db.QueryContext(ctx, sqlQuery, likeQuery, likeQuery, likeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		var metadataRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for node %s: %w", node.ID, err)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// Propagate implements Personalized PageRank (PPR) for wisdom nodes.
// It propagates relevance signals from seed nodes across the graph.
// Optimized with parallel edge fetching and OTel tracing.
func (c *Cortex) Propagate(ctx context.Context, seedIDs []string, alpha float64, iterations int) (map[string]float64, error) {
	_, span := observability.Tracer.Start(ctx, "Cortex.Propagate")
	defer span.End()

	if len(seedIDs) == 0 {
		return nil, nil
	}

	// Initialize scores
	scores := make(map[string]float64)
	initialScore := 1.0 / float64(len(seedIDs))
	for _, id := range seedIDs {
		scores[id] = initialScore
	}

	teleport := (1 - alpha) / float64(len(seedIDs))

	for i := 0; i < iterations; i++ {
		iterCtx, iterSpan := observability.Tracer.Start(ctx, fmt.Sprintf("Iteration.%d", i))

		newScores := make(map[string]float64)
		var mu sync.Mutex

		// 1. Collect current nodes with non-zero scores
		var currentNodes []string
		for id, score := range scores {
			if score > 0 {
				currentNodes = append(currentNodes, id)
			}
		}

		// 2. Fetch edges in batches using concurrency if set is large
		const batchSize = 100
		g, gCtx := errgroup.WithContext(iterCtx)

		for j := 0; j < len(currentNodes); j += batchSize {
			end := j + batchSize
			if end > len(currentNodes) {
				end = len(currentNodes)
			}
			batch := currentNodes[j:end]

			g.Go(func() error {
				edges, err := c.GetEdges(gCtx, batch)
				if err != nil {
					return err
				}

				mu.Lock()
				defer mu.Unlock()
				for _, edge := range edges {
					contribution := scores[edge.SourceID] * alpha * edge.Weight
					newScores[edge.TargetID] += contribution
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			iterSpan.End()
			return nil, fmt.Errorf("failed to fetch edges in iteration %d: %w", i, err)
		}

		// 3. Apply teleportation back to seeds
		for _, seed := range seedIDs {
			newScores[seed] += teleport
		}

		// Update scores for next iteration
		scores = newScores
		iterSpan.End()
	}

	return scores, nil
}
