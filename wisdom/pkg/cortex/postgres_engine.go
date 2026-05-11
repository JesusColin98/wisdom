package cortex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresEngine implements StorageEngine using PostgreSQL (optimized for Neon/Supabase).
type PostgresEngine struct {
	db *sql.DB
}

func NewPostgresEngine(connStr string) (*PostgresEngine, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}
	return &PostgresEngine{db: db}, nil
}

func (e *PostgresEngine) Close() error {
	return e.db.Close()
}

// NodeStore implementation

func (e *PostgresEngine) PutNode(ctx context.Context, node *Node) error {
	metadataJSON, _ := json.Marshal(node.Metadata)
	linksJSON, _ := json.Marshal(node.ExternalLinks)

	query := `
		INSERT INTO nodes (id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			content = EXCLUDED.content,
			entity_class = EXCLUDED.entity_class,
			author = EXCLUDED.author,
			source_type = EXCLUDED.source_type,
			source_ref = EXCLUDED.source_ref,
			metadata = EXCLUDED.metadata,
			confidence_score = EXCLUDED.confidence_score,
			impact_score = EXCLUDED.impact_score,
			stratum = EXCLUDED.stratum,
			source_mime_type = EXCLUDED.source_mime_type,
			external_links = EXCLUDED.external_links,
			superseded_by_id = EXCLUDED.superseded_by_id,
			valid_from = EXCLUDED.valid_from,
			valid_until = EXCLUDED.valid_until,
			repetition_count = EXCLUDED.repetition_count,
			easiness_factor = EXCLUDED.easiness_factor,
			next_review_at = EXCLUDED.next_review_at,
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

	_, err := e.db.ExecContext(ctx, query,
		node.ID, node.Content, node.EntityClass, node.Author, node.SourceType,
		node.SourceRef, node.NamespaceID, metadataJSON, node.ConfidenceScore,
		node.ImpactScore, node.Stratum, node.SourceMimeType, linksJSON, supersededBy, node.ValidFrom, node.ValidUntil,
		node.RepetitionCount, node.EasinessFactor, nextReviewAt,
	)
	return err
}

func (e *PostgresEngine) GetNode(ctx context.Context, id string) (*Node, error) {
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, created_at, updated_at FROM nodes WHERE id = $1`
	row := e.db.QueryRowContext(ctx, query, id)

	var node Node
	var metadataRaw, linksRaw []byte
	var supersededByID sql.NullString
	var validFrom, validUntil, nextReviewAt sql.NullTime

	err := row.Scan(
		&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
		&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
		&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw, &supersededByID, &validFrom, &validUntil,
		&node.RepetitionCount, &node.EasinessFactor, &nextReviewAt,
		&node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		return nil, err
	}

	if supersededByID.Valid { node.SupersededByID = supersededByID.String }
	if validFrom.Valid { node.ValidFrom = validFrom.Time }
	if validUntil.Valid { node.ValidUntil = validUntil.Time }
	if nextReviewAt.Valid { node.NextReviewAt = nextReviewAt.Time }

	json.Unmarshal(metadataRaw, &node.Metadata)
	json.Unmarshal(linksRaw, &node.ExternalLinks)

	return &node, nil
}

func (e *PostgresEngine) DeleteNode(ctx context.Context, id string) error {
	_, err := e.db.ExecContext(ctx, "DELETE FROM nodes WHERE id = $1", id)
	return err
}

func (e *PostgresEngine) SearchNodes(ctx context.Context, query string) ([]Node, error) {
	sqlQuery := `
		SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, created_at, updated_at
		FROM nodes
		WHERE content ILIKE $1 OR id ILIKE $1 OR entity_class ILIKE $1
	`
	likeQuery := "%" + query + "%"
	rows, err := e.db.QueryContext(ctx, sqlQuery, likeQuery)
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
		if err != nil { return nil, err }
		json.Unmarshal(metadataRaw, &node.Metadata)
		json.Unmarshal(linksRaw, &node.ExternalLinks)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (e *PostgresEngine) ListNodes(ctx context.Context, filter map[string]any) ([]Node, error) {
	namespaceID, _ := filter["namespace_id"].(string)
	query := `SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, created_at, updated_at FROM nodes WHERE namespace_id = $1`
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
		if err != nil { return nil, err }
		json.Unmarshal(metadataRaw, &node.Metadata)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (e *PostgresEngine) PruneNodes(ctx context.Context, lambda float64, threshold float64) (int, error) {
	decayQuery := `
		UPDATE nodes 
		SET confidence_score = confidence_score * exp(-$1 * EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - updated_at))),
		    updated_at = CURRENT_TIMESTAMP
		WHERE confidence_score > 0
	`
	_, err := e.db.ExecContext(ctx, decayQuery, lambda)
	if err != nil { return 0, err }

	deleteQuery := `DELETE FROM nodes WHERE confidence_score < $1 AND impact_score < $1`
	res, err := e.db.ExecContext(ctx, deleteQuery, threshold)
	if err != nil { return 0, err }

	rowsAffected, _ := res.RowsAffected()
	return int(rowsAffected), nil
}

func (e *PostgresEngine) ListDueNodes(ctx context.Context, namespaceID string, limit int) ([]Node, error) {
	query := `
		SELECT id, content, entity_class, author, source_type, source_ref, namespace_id, metadata, confidence_score, impact_score, stratum, source_mime_type, external_links, superseded_by_id, valid_from, valid_until, repetition_count, easiness_factor, next_review_at, created_at, updated_at
		FROM nodes
		WHERE namespace_id = $1 AND next_review_at <= CURRENT_TIMESTAMP
		ORDER BY next_review_at ASC
		LIMIT $2
	`
	rows, err := e.db.QueryContext(ctx, query, namespaceID, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	nodes := []Node{}
	for rows.Next() {
		var node Node
		var metadataRaw, linksRaw []byte
		var supersededByID sql.NullString
		var validFrom, validUntil, nextReviewAt sql.NullTime

		err := rows.Scan(
			&node.ID, &node.Content, &node.EntityClass, &node.Author, &node.SourceType,
			&node.SourceRef, &node.NamespaceID, &metadataRaw, &node.ConfidenceScore,
			&node.ImpactScore, &node.Stratum, &node.SourceMimeType, &linksRaw, &supersededByID, &validFrom, &validUntil,
			&node.RepetitionCount, &node.EasinessFactor, &nextReviewAt,
			&node.CreatedAt, &node.UpdatedAt,
		)
		if err != nil { return nil, err }

		if supersededByID.Valid { node.SupersededByID = supersededByID.String }
		if validFrom.Valid { node.ValidFrom = validFrom.Time }
		if validUntil.Valid { node.ValidUntil = validUntil.Time }
		if nextReviewAt.Valid { node.NextReviewAt = nextReviewAt.Time }

		json.Unmarshal(metadataRaw, &node.Metadata)
		json.Unmarshal(linksRaw, &node.ExternalLinks)
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (e *PostgresEngine) UpdateConfidence(ctx context.Context, nodeID string, delta float64) error {
	query := `UPDATE nodes SET confidence_score = LEAST(1.0, GREATEST(0.0, confidence_score + $1)), updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := e.db.ExecContext(ctx, query, delta, nodeID)
	return err
}

func (e *PostgresEngine) UpdateImpact(ctx context.Context, nodeID string, delta float64) error {
	query := `UPDATE nodes SET impact_score = LEAST(1.0, GREATEST(0.0, impact_score + $1)), updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := e.db.ExecContext(ctx, query, delta, nodeID)
	return err
}

func (e *PostgresEngine) MoveToCold(ctx context.Context, nodeID string) error {
	query := `UPDATE nodes SET stratum = 'COLD', updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := e.db.ExecContext(ctx, query, nodeID)
	return err
}

func (e *PostgresEngine) RecallToHot(ctx context.Context, nodeID string) error {
	query := `UPDATE nodes SET stratum = 'HOT', updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := e.db.ExecContext(ctx, query, nodeID)
	return err
}

// VectorStore implementation

func (e *PostgresEngine) PutVector(ctx context.Context, nodeID string, embedding []float32, modelVersion string) error {
	vectorStr := floatsToVectorString(embedding)
	query := `
		INSERT INTO vectors (node_id, embedding, model_version, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT(node_id) DO UPDATE SET
			embedding = EXCLUDED.embedding,
			model_version = EXCLUDED.model_version,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := e.db.ExecContext(ctx, query, nodeID, vectorStr, modelVersion)
	return err
}

func (e *PostgresEngine) GetVector(ctx context.Context, nodeID string) ([]float32, string, error) {
	query := `SELECT embedding::text, model_version FROM vectors WHERE node_id = $1`
	row := e.db.QueryRowContext(ctx, query, nodeID)

	var vectorStr, modelVersion string
	if err := row.Scan(&vectorStr, &modelVersion); err != nil {
		if err == sql.ErrNoRows { return nil, "", nil }
		return nil, "", err
	}

	return vectorStringToFloats(vectorStr), modelVersion, nil
}

func (e *PostgresEngine) SearchVectors(ctx context.Context, embedding []float32, topK int) ([]ScoredNode, error) {
	vectorStr := floatsToVectorString(embedding)
	// Using cosine distance (<=>) from pgvector. Score = 1 - distance.
	query := `
		SELECT n.id, n.content, n.entity_class, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.confidence_score, n.impact_score, n.stratum, n.source_mime_type, n.external_links, n.created_at, n.updated_at, (1 - (v.embedding <=> $1)) as score
		FROM nodes n
		JOIN vectors v ON n.id = v.node_id
		ORDER BY v.embedding <=> $1
		LIMIT $2
	`
	rows, err := e.db.QueryContext(ctx, query, vectorStr, topK)
	if err != nil { return nil, err }
	defer rows.Close()

	results := []ScoredNode{}
	for rows.Next() {
		var sn ScoredNode
		var metadataRaw, linksRaw []byte
		err := rows.Scan(
			&sn.ID, &sn.Content, &sn.EntityClass, &sn.Author, &sn.SourceType,
			&sn.SourceRef, &sn.NamespaceID, &metadataRaw, &sn.ConfidenceScore,
			&sn.ImpactScore, &sn.Stratum, &sn.SourceMimeType, &linksRaw, &sn.CreatedAt, &sn.UpdatedAt, &sn.Score,
		)
		if err != nil { return nil, err }
		json.Unmarshal(metadataRaw, &sn.Metadata)
		json.Unmarshal(linksRaw, &sn.ExternalLinks)
		results = append(results, sn)
	}
	return results, nil
}

func (e *PostgresEngine) DeleteVector(ctx context.Context, nodeID string) error {
	_, err := e.db.ExecContext(ctx, "DELETE FROM vectors WHERE node_id = $1", nodeID)
	return err
}

// GraphStore implementation

func (e *PostgresEngine) LinkNodes(ctx context.Context, link *Link) error {
	query := `
		INSERT INTO links (source_id, target_id, relation_type, weight)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(source_id, target_id, relation_type) DO UPDATE SET
			weight = EXCLUDED.weight
	`
	_, err := e.db.ExecContext(ctx, query, link.SourceID, link.TargetID, link.RelationType, link.Weight)
	return err
}

func (e *PostgresEngine) GetNeighbors(ctx context.Context, nodeID string) ([]*Node, error) {
	query := `
		SELECT n.id, n.content, n.author, n.source_type, n.source_ref, n.namespace_id, n.metadata, n.created_at, n.updated_at
		FROM nodes n
		JOIN links l ON n.id = l.target_id
		WHERE l.source_id = $1
	`
	rows, err := e.db.QueryContext(ctx, query, nodeID)
	if err != nil { return nil, err }
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
		if err != nil { return nil, err }
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

func (e *PostgresEngine) GetEdges(ctx context.Context, sourceIDs []string) ([]Edge, error) {
	if len(sourceIDs) == 0 { return nil, nil }
	
	placeholders := make([]string, len(sourceIDs))
	args := make([]any, len(sourceIDs))
	for i, id := range sourceIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`SELECT source_id, target_id, weight FROM links WHERE source_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var edge Edge
		if err := rows.Scan(&edge.SourceID, &edge.TargetID, &edge.Weight); err != nil { return nil, err }
		edges = append(edges, edge)
	}
	return edges, nil
}

func (e *PostgresEngine) ListEdges(ctx context.Context) ([]Link, error) {
	query := `SELECT source_id, target_id, relation_type, weight, created_at FROM links`
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil { return nil, err }
	defer rows.Close()

	links := []Link{}
	for rows.Next() {
		var link Link
		err := rows.Scan(&link.SourceID, &link.TargetID, &link.RelationType, &link.Weight, &link.CreatedAt)
		if err != nil { return nil, err }
		links = append(links, link)
	}
	return links, nil
}

func (e *PostgresEngine) ResolvePointer(ctx context.Context, pointer string) (string, error) {
	// 1. Synonym check
	query := `
		SELECT target_id 
		FROM links 
		WHERE source_id = $1 AND relation_type = 'SYNONYM_OF'
		ORDER BY weight DESC LIMIT 1
	`
	var id string
	err := e.db.QueryRowContext(ctx, query, pointer).Scan(&id)
	if err == nil { return id, nil }

	// 2. Direct ID check
	err = e.db.QueryRowContext(ctx, `SELECT id FROM nodes WHERE id = $1`, pointer).Scan(&id)
	if err == nil { return id, nil }

	return "", fmt.Errorf("could not resolve pointer: %s", pointer)
}

// SessionStore implementation

func (e *PostgresEngine) AddLog(ctx context.Context, sessionID, role, content string) error {
	query := `INSERT INTO session_logs (session_id, role, content) VALUES ($1, $2, $3)`
	_, err := e.db.ExecContext(ctx, query, sessionID, role, content)
	return err
}

func (e *PostgresEngine) GetLogs(ctx context.Context, sessionID string) ([]Interaction, error) {
	query := `SELECT role, content FROM session_logs WHERE session_id = $1 ORDER BY log_id ASC`
	rows, err := e.db.QueryContext(ctx, query, sessionID)
	if err != nil { return nil, err }
	defer rows.Close()

	var logs []Interaction
	for rows.Next() {
		var i Interaction
		if err := rows.Scan(&i.Role, &i.Content); err != nil { return nil, err }
		logs = append(logs, i)
	}
	return logs, nil
}

func (e *PostgresEngine) ClearLogs(ctx context.Context, sessionID string) error {
	query := `DELETE FROM session_logs WHERE session_id = $1`
	_, err := e.db.ExecContext(ctx, query, sessionID)
	return err
}

func (e *PostgresEngine) GetInactiveSessions(ctx context.Context, olderThan time.Duration) ([]string, error) {
	query := `
		SELECT session_id
		FROM session_logs
		GROUP BY session_id
		HAVING MAX(created_at) < (CURRENT_TIMESTAMP - $1::interval)
	`
	interval := fmt.Sprintf("%d seconds", int(olderThan.Seconds()))
	rows, err := e.db.QueryContext(ctx, query, interval)
	if err != nil { return nil, err }
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil { return nil, err }
		sessions = append(sessions, id)
	}
	return sessions, nil
}

// Helpers

func floatsToVectorString(floats []float32) string {
	strs := make([]string, len(floats))
	for i, f := range floats {
		strs[i] = fmt.Sprintf("%f", f)
	}
	return "[" + strings.Join(strs, ",") + "]"
}

func vectorStringToFloats(s string) []float32 {
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, ",")
	floats := make([]float32, len(parts))
	for i, p := range parts {
		fmt.Sscanf(p, "%f", &floats[i])
	}
	return floats
}
