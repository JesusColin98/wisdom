package cortex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
)

// PostgresEngine implements StorageEngine using PostgreSQL with the Universal Graph Schema.
type PostgresEngine struct {
	db *sql.DB
}

// NewPostgresEngine initializes the connection.
func NewPostgresEngine(connStr string) (*PostgresEngine, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}
	return &PostgresEngine{db: db}, nil
}

// Close closes the database connection.
func (e *PostgresEngine) Close() error {
	return e.db.Close()
}

// Memorize inserts or updates a Node in the database.
func (e *PostgresEngine) Memorize(ctx context.Context, node *Node) error {
	payloadJSON, err := json.Marshal(node.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal node payload: %w", err)
	}

	query := `
		INSERT INTO nodes (id, type, payload, confidence, requires_human, ttl, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			payload = EXCLUDED.payload,
			confidence = EXCLUDED.confidence,
			requires_human = EXCLUDED.requires_human,
			ttl = EXCLUDED.ttl,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = e.db.ExecContext(ctx, query,
		node.ID, node.Type, payloadJSON, node.Confidence, node.RequiresHuman, node.TTL,
	)
	return err
}

// GetNode fetches a single Node by its ID.
func (e *PostgresEngine) GetNode(ctx context.Context, id string) (*Node, error) {
	query := `
		SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at
		FROM nodes
		WHERE id = $1
	`
	row := e.db.QueryRowContext(ctx, query, id)

	var node Node
	var payloadRaw []byte
	var ttl sql.NullTime

	err := row.Scan(
		&node.ID, &node.Type, &payloadRaw, &node.Confidence, &node.RequiresHuman,
		&ttl, &node.CreatedAt, &node.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}

	if ttl.Valid {
		node.TTL = &ttl.Time
	}

	if err := json.Unmarshal(payloadRaw, &node.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &node, nil
}

// QueryFacts queries nodes of type Fact where the JSONB payload contains the given metadata keys/values.
// This is a simplified version; complex queries might require specific GIN operators.
func (e *PostgresEngine) QueryFacts(ctx context.Context, metadataFilters map[string]string) ([]*Node, error) {
	// Start with the base query for Facts
	query := `SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at FROM nodes WHERE type = 'Fact'`
	args := []any{}

	// If there are metadata filters, construct the JSONB containment query
	if len(metadataFilters) > 0 {
		filterJSON, err := json.Marshal(metadataFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata filters: %w", err)
		}
		query += ` AND payload @> $1`
		args = append(args, filterJSON)
	}

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []*Node
	for rows.Next() {
		var node Node
		var payloadRaw []byte
		var ttl sql.NullTime
		if err := rows.Scan(
			&node.ID, &node.Type, &payloadRaw, &node.Confidence, &node.RequiresHuman,
			&ttl, &node.CreatedAt, &node.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if ttl.Valid {
			node.TTL = &ttl.Time
		}
		if err := json.Unmarshal(payloadRaw, &node.Payload); err != nil {
			return nil, err
		}
		facts = append(facts, &node)
	}
	return facts, nil
}

// AddEdge creates a relationship between two nodes.
func (e *PostgresEngine) AddEdge(ctx context.Context, edge *Edge) error {
	query := `
		INSERT INTO edges (source_id, target_id, relation)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_id, target_id, relation) DO NOTHING
	`
	_, err := e.db.ExecContext(ctx, query, edge.SourceID, edge.TargetID, edge.Relation)
	return err
}

// GetNodes fetches multiple Nodes by their IDs in a single batch.
func (e *PostgresEngine) GetNodes(ctx context.Context, ids []string) ([]*Node, error) {
	if len(ids) == 0 {
		return []*Node{}, nil
	}

	query := `
		SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at
		FROM nodes
		WHERE id = ANY($1)
	`
	rows, err := e.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		var node Node
		var payloadRaw []byte
		var ttl sql.NullTime
		if err := rows.Scan(
			&node.ID, &node.Type, &payloadRaw, &node.Confidence, &node.RequiresHuman,
			&ttl, &node.CreatedAt, &node.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if ttl.Valid {
			node.TTL = &ttl.Time
		}
		if err := json.Unmarshal(payloadRaw, &node.Payload); err != nil {
			return nil, err
		}
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

// Recall retrieves a Node and its direct incoming and outgoing edges, plus the neighbor nodes.
func (e *PostgresEngine) Recall(ctx context.Context, id string) (*CognitionResponse, error) {
	center, err := e.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}
	if center == nil {
		return nil, nil // Node not found
	}

	response := &CognitionResponse{
		Center: center,
	}

	// Fetch Outgoing Edges
	outQuery := `SELECT source_id, target_id, relation, created_at FROM edges WHERE source_id = $1`
	outRows, err := e.db.QueryContext(ctx, outQuery, id)
	if err != nil {
		return nil, err
	}
	defer outRows.Close()

	neighborIDs := make(map[string]bool)
	for outRows.Next() {
		var edge Edge
		if err := outRows.Scan(&edge.SourceID, &edge.TargetID, &edge.Relation, &edge.CreatedAt); err != nil {
			return nil, err
		}
		response.OutEdges = append(response.OutEdges, &edge)
		neighborIDs[edge.TargetID] = true
	}

	// Fetch Incoming Edges
	inQuery := `SELECT source_id, target_id, relation, created_at FROM edges WHERE target_id = $1`
	inRows, err := e.db.QueryContext(ctx, inQuery, id)
	if err != nil {
		return nil, err
	}
	defer inRows.Close()

	for inRows.Next() {
		var edge Edge
		if err := inRows.Scan(&edge.SourceID, &edge.TargetID, &edge.Relation, &edge.CreatedAt); err != nil {
			return nil, err
		}
		response.InEdges = append(response.InEdges, &edge)
		neighborIDs[edge.SourceID] = true
	}

	// Fetch Neighbor Nodes
	if len(neighborIDs) > 0 {
		ids := make([]string, 0, len(neighborIDs))
		for nid := range neighborIDs {
			ids = append(ids, nid)
		}
		neighbors, err := e.GetNodes(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch neighbors: %w", err)
		}
		response.Nodes = neighbors
	}

	return response, nil
}

// GetAllNodes fetches all nodes from the database.
func (e *PostgresEngine) GetAllNodes(ctx context.Context) ([]*Node, error) {
	query := `SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at FROM nodes`
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		var node Node
		var payloadRaw []byte
		var ttl sql.NullTime
		if err := rows.Scan(
			&node.ID, &node.Type, &payloadRaw, &node.Confidence, &node.RequiresHuman,
			&ttl, &node.CreatedAt, &node.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if ttl.Valid {
			node.TTL = &ttl.Time
		}
		if err := json.Unmarshal(payloadRaw, &node.Payload); err != nil {
			return nil, err
		}
		nodes = append(nodes, &node)
	}
	return nodes, nil
}

// GetAllEdges fetches all edges from the database.
func (e *PostgresEngine) GetAllEdges(ctx context.Context) ([]*Edge, error) {
	query := `SELECT source_id, target_id, relation, created_at FROM edges`
	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []*Edge
	for rows.Next() {
		var edge Edge
		if err := rows.Scan(&edge.SourceID, &edge.TargetID, &edge.Relation, &edge.CreatedAt); err != nil {
			return nil, err
		}
		edges = append(edges, &edge)
	}
	return edges, nil
}
