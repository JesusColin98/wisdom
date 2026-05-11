package cortex

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/wisdom/pkg/observability"
	"golang.org/x/sync/errgroup"
)

// SQLiteEngine implements StorageEngine using SQLite.
type SQLiteEngine struct {
	db        *sql.DB
	substrate VectorSubstrate
	indexPath string
	trie      *SCGTrie
}

func NewSQLiteEngine(db *sql.DB, indexPath string) *SQLiteEngine {
	return &SQLiteEngine{
		db:        db,
		indexPath: indexPath,
		trie:      NewSCGTrie(),
	}
}

func (e *SQLiteEngine) Close() error {
	return e.db.Close()
}

// NodeStore implementation

func (e *SQLiteEngine) PutNode(ctx context.Context, node *Node) error {
	metadataJSON, err := json.Marshal(node.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	linksJSON, err := json.Marshal(node.ExternalLinks)
	if err != nil {
		return fmt.Errorf("failed to marshal external links: %w", err)
	}

	query := `
		INSERT INTO nodes (id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			entity_class = excluded.entity_class,
			author = excluded.author,
			source_type = excluded.source_type,
			source_ref = excluded.source_ref,
			metadata = excluded.metadata,
			confidence_score = excluded.confidence_score,
			impact_score = excluded.impact_score,
			stratum = excluded.stratum,
			source_mime_type = excluded.source_mime_type,
			external_links = excluded.external_links,
			superseded_by_id = excluded.superseded_by_id,
			valid_from = excluded.valid_from,
			valid_until = excluded.valid_until,
			repetition_count = excluded.repetition_count,
			easiness_factor = excluded.easiness_factor,
			next_review_at = excluded.next_review_at,
			updated_at = CURRENT_TIMESTAMP
	`
	var supersededBy sql.NullString
	if node.SupersededByID != "" {
		supersededBy = sql.NullString{String: node.SupersededByID, Valid: true}
	}

	var nextReviewAt sql.NullTime
	if !node.NextReviewAt.IsZero() {
		nextReviewAt = sql.NullTime{Time: node.NextReviewAt, Valid: true}
	}

	if node.Stratum == "" {
		node.Stratum = "HOT"
	}
	if node.SourceMimeType == "" {
		node.SourceMimeType = "text/plain"
	}

	_, err = e.db.ExecContext(ctx, query,
		node.ID, node.Content, node.EntityClass, node.Author, node.SourceType,
		node.SourceRef, node.NamespaceID, metadataJSON, node.ConfidenceScore,
		node.ImpactScore, node.Stratum, node.SourceMimeType, linksJSON, supersededBy, node.ValidFrom, node.ValidUntil,
		node.RepetitionCount, node.EasinessFactor, nextReviewAt,
	)
	if err != nil {
		return err
	}

	// Update Trie
	e.trie.Insert(node.Content, node.ID)
	e.trie.Insert(node.ID, node.ID)

	return nil
}

func (e *SQLiteEngine) GetNode(ctx context.Context, id string) (*Node, error) {
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, created_at, updated_at FROM nodes WHERE id = ?`
	row := e.db.QueryRowContext(ctx, query, id)

	var node Node
	var metadataRaw, linksRaw []byte
	var supersededByID, validFrom, validUntil sql.NullString
	var nextReviewAt sql.NullTime

	err := row.Scan(
		&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
		&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
		&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw, &supersededByID, &validFrom, &validUntil,
		&node.RepetitionCount, &node.EasinessFactor, &nextReviewAt,
		&node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if supersededByID.Valid { node.SupersededByID = supersededByID.String }
	if validFrom.Valid { node.ValidFrom, _ = time.Parse(time.RFC3339, validFrom.String) }
	if validUntil.Valid { node.ValidUntil, _ = time.Parse(time.RFC3339, validUntil.String) }
	if nextReviewAt.Valid { node.NextReviewAt = nextReviewAt.Time }

	if err := json.Unmarshal(metadataRaw, &node.Metadata); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(linksRaw, &node.ExternalLinks); err != nil {
		return nil, err
	}

	return &node, nil
}

func (e *SQLiteEngine) DeleteNode(ctx context.Context, id string) error {
	_, err := e.db.ExecContext(ctx, "DELETE FROM nodes WHERE id = ?", id)
	return err
}

func (e *SQLiteEngine) SearchNodes(ctx context.Context, query string) ([]Node, error) {
	sqlQuery := `
		SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, created_at, updated_at
		FROM nodes
		WHERE content LIKE ? OR id LIKE ? OR entity_class LIKE ?
	`
	likeQuery := "%" + query + "%"
	rows, err := e.db.QueryContext(ctx, sqlQuery, likeQuery, likeQuery, likeQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var metadataRaw, linksRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(metadataRaw, &node.Metadata)
		json.Unmarshal(linksRaw, &node.ExternalLinks)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (e *SQLiteEngine) ListNodes(ctx context.Context, filter map[string]any) ([]Node, error) {
	// Simple implementation for now, prioritizing namespace_id
	namespaceID, _ := filter["namespace_id"].(string)
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, created_at, updated_at FROM nodes WHERE namespace_id = ?`
	rows, err := e.db.QueryContext(ctx, query, namespaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var metadataRaw []byte
		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.ImpactScore, &node.Stratum, &node.SourceMimeType,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(metadataRaw, &node.Metadata)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (e *SQLiteEngine) PruneNodes(ctx context.Context, lambda float64, threshold float64) (int, error) {
	decayQuery := `
		UPDATE nodes 
		SET confidence_score = confidence_score * exp(-? * (strftime('%s', 'now') - strftime('%s', updated_at))),
		    updated_at = CURRENT_TIMESTAMP
		WHERE confidence_score > 0
	`
	_, err := e.db.ExecContext(ctx, decayQuery, lambda)
	if err != nil {
		return 0, err
	}

	deleteQuery := `DELETE FROM nodes WHERE confidence_score < ? AND impact_score < ?`
	res, err := e.db.ExecContext(ctx, deleteQuery, threshold, threshold)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := res.RowsAffected()
	return int(rowsAffected), nil
}

// VectorStore implementation

func (e *SQLiteEngine) PutVector(ctx context.Context, nodeID string, embedding []float32, modelVersion string) error {
	blob := floatsToBytes(embedding)
	query := `
		INSERT INTO vectors (node_id, embedding, model_version, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(node_id) DO UPDATE SET
			embedding = excluded.embedding,
			model_version = excluded.model_version,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := e.db.ExecContext(ctx, query, nodeID, blob, modelVersion)
	if err != nil {
		return err
	}

	if e.substrate != nil {
		_ = e.substrate.Add(ctx, nodeID, embedding)
	}

	return nil
}

func (e *SQLiteEngine) GetVector(ctx context.Context, nodeID string) ([]float32, string, error) {
	query := `SELECT embedding, model_version FROM vectors WHERE node_id = ?`
	row := e.db.QueryRowContext(ctx, query, nodeID)

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

func (e *SQLiteEngine) SearchVectors(ctx context.Context, embedding []float32, topK int) ([]ScoredNode, error) {
	if e.substrate != nil {
		return e.substrate.Search(ctx, embedding, topK)
	}
	return e.linearVectorSearch(ctx, embedding, topK)
}

func (e *SQLiteEngine) DeleteVector(ctx context.Context, nodeID string) error {
	_, err := e.db.ExecContext(ctx, "DELETE FROM vectors WHERE node_id = ?", nodeID)
	return err
}

func (e *SQLiteEngine) linearVectorSearch(ctx context.Context, queryEmbedding []float32, topK int) ([]ScoredNode, error) {
	query := `
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.impact_score, n.stratum, n.source_mime_type, n.external_links, n.created_at, n.updated_at, v.embedding
		FROM nodes n
		JOIN vectors v ON n.id = v.node_id
	`
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []ScoredNode{}
	for rows.Next() {
		var sn ScoredNode
		var metadataRaw, linksRaw []byte
		var embeddingRaw []byte
		err := rows.Scan(
			&sn.ID, &sn.Content, &sn.EntityClass, &sn.Author, &sn.SourceType,
			&sn.SourceRef, &sn.NamespaceID, &metadataRaw, &sn.ConfidenceScore,
			&sn.ImpactScore, &sn.Stratum, &sn.SourceMimeType, &linksRaw, &sn.CreatedAt, &sn.UpdatedAt, &embeddingRaw,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(metadataRaw, &sn.Metadata)
		json.Unmarshal(linksRaw, &sn.ExternalLinks)

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

// GraphStore implementation

func (e *SQLiteEngine) LinkNodes(ctx context.Context, link *Link) error {
	query := `
		INSERT INTO links (source_id, target_id, relation_type, weight)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(source_id, target_id, relation_type) DO UPDATE SET
			weight = excluded.weight
	`
	_, err := e.db.ExecContext(ctx, query, link.SourceID, link.TargetID, link.RelationType, link.Weight)
	return err
}

func (e *SQLiteEngine) GetNeighbors(ctx context.Context, nodeID string) ([]*Node, error) {
	query := `
		SELECT n.id, n.content, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.created_at, n.updated_at
		FROM nodes n
		JOIN links l ON n.id = l.target_id
		WHERE l.source_id = ?
	`
	rows, err := e.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []*Node{}
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
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

func (e *SQLiteEngine) GetEdges(ctx context.Context, sourceIDs []string) ([]Edge, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}

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
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var edge Edge
		if err := rows.Scan(&edge.SourceID, &edge.TargetID, &edge.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, nil
}

func (e *SQLiteEngine) ListEdges(ctx context.Context) ([]Link, error) {
	query := `SELECT source_id, target_id, relation_type, weight, created_at FROM links`
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []Link{}
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

// SessionStore implementation

func (e *SQLiteEngine) AddLog(ctx context.Context, sessionID, role, content string) error {
	query := `INSERT INTO session_logs (session_id, role, content) VALUES (?, ?, ?)`
	_, err := e.db.ExecContext(ctx, query, sessionID, role, content)
	return err
}

func (e *SQLiteEngine) GetLogs(ctx context.Context, sessionID string) ([]Interaction, error) {
	query := `SELECT role, content FROM session_logs WHERE session_id = ? ORDER BY log_id ASC`
	rows, err := e.db.QueryContext(ctx, query, sessionID)
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

func (e *SQLiteEngine) ClearLogs(ctx context.Context, sessionID string) error {
	query := `DELETE FROM session_logs WHERE session_id = ?`
	_, err := e.db.ExecContext(ctx, query, sessionID)
	return err
}

func (e *SQLiteEngine) ListDueNodes(ctx context.Context, namespaceID string, limit int) ([]Node, error) {
	query := `
		SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, created_at, updated_at
		FROM nodes
		WHERE namespace_id = ? AND next_review_at <= CURRENT_TIMESTAMP
		ORDER BY next_review_at ASC
		LIMIT ?
	`
	rows, err := e.db.QueryContext(ctx, query, namespaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var metadataRaw, linksRaw []byte
		var supersededByID, validFrom, validUntil sql.NullString
		var nextReviewAt sql.NullTime

		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw, &supersededByID, &validFrom, &validUntil,
			&node.RepetitionCount, &node.EasinessFactor, &nextReviewAt,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if supersededByID.Valid { node.SupersededByID = supersededByID.String }
		if validFrom.Valid { node.ValidFrom, _ = time.Parse(time.RFC3339, validFrom.String) }
		if validUntil.Valid { node.ValidUntil, _ = time.Parse(time.RFC3339, validUntil.String) }
		if nextReviewAt.Valid { node.NextReviewAt = nextReviewAt.Time }

		json.Unmarshal(metadataRaw, &node.Metadata)
		json.Unmarshal(linksRaw, &node.ExternalLinks)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (e *SQLiteEngine) UpdateConfidence(ctx context.Context, nodeID string, delta float64) error {
	query := `UPDATE nodes SET confidence_score = MIN(1.0, MAX(0.0, confidence_score + ?)), updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := e.db.ExecContext(ctx, query, delta, nodeID)
	return err
}

func (e *SQLiteEngine) UpdateImpact(ctx context.Context, nodeID string, delta float64) error {
	query := `UPDATE nodes SET impact_score = MIN(1.0, MAX(0.0, impact_score + ?)), updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := e.db.ExecContext(ctx, query, delta, nodeID)
	return err
}

func (e *SQLiteEngine) MoveToCold(ctx context.Context, nodeID string) error {
	query := `UPDATE nodes SET stratum = 'COLD', updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := e.db.ExecContext(ctx, query, nodeID)
	return err
}

func (e *SQLiteEngine) RecallToHot(ctx context.Context, nodeID string) error {
	query := `UPDATE nodes SET stratum = 'HOT', updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := e.db.ExecContext(ctx, query, nodeID)
	return err
}

func (e *SQLiteEngine) PromoteSubstrate(ctx context.Context) error {
	if _, ok := e.substrate.(*RPForestSubstrate); ok {
		return nil
	}

	observability.Logger.Info("Promoting substrate to RPForest Tier")

	query := `SELECT node_id, embedding FROM vectors`
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	forest := NewRPForestSubstrate(nil, 10, 768) // storage will be set below
	// forest.storage is needed for GetVector during split, but we can't easily set it to *Cortex here without circular deps
	// Actually RPForestSubstrate.storage is *Cortex. 
	// We'll need to refactor substrate.go to use StorageEngine or a narrower interface.
	
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			continue
		}
		vec := bytesToFloats(blob)
		if len(vec) > 0 {
			forest.Dim = len(vec)
			_ = forest.Add(ctx, id, vec)
		}
	}

	e.substrate = forest
	return forest.Save(e.indexPath)
}

func (e *SQLiteEngine) scoreSpecificNodes(ctx context.Context, queryEmbedding []float32, ids []string, topK int) ([]ScoredNode, error) {
	if len(ids) == 0 {
		return nil, nil
	}

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
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.impact_score, n.stratum, n.source_mime_type, n.external_links, n.created_at, n.updated_at, v.embedding
		FROM nodes n
		JOIN vectors v ON n.id = v.node_id
		WHERE n.id IN (%s)
	`, placeholders)

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []ScoredNode{}
	for rows.Next() {
		var sn ScoredNode
		var metadataRaw, linksRaw []byte
		var embeddingRaw []byte
		err := rows.Scan(
			&sn.ID, &sn.Content, &sn.EntityClass, &sn.Author, &sn.SourceType,
			&sn.SourceRef, &sn.NamespaceID, &metadataRaw, &sn.ConfidenceScore,
			&sn.ImpactScore, &sn.Stratum, &sn.SourceMimeType, &linksRaw, &sn.CreatedAt, &sn.UpdatedAt, &embeddingRaw,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(metadataRaw, &sn.Metadata)
		json.Unmarshal(linksRaw, &sn.ExternalLinks)

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


// Helper methods from storage.go

func floatsToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func bytesToFloats(b []byte) []float32 {
	floats := make([]float32, len(b)/4)
	for i := range floats {
		floats[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return floats
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
