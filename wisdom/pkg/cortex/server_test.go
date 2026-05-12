package cortex

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pb "github.com/google/wisdom/pkg/cortex/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestServer_Memorize(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	engine := &PostgresEngine{db: db}
	server := NewServer(engine)

	payload, _ := structpb.NewStruct(map[string]any{"key": "value"})

	req := &pb.IngestRequest{
		Id:            "test-server-1",
		Type:          "Concept",
		Payload:       payload,
		Confidence:    0.95,
		RequiresHuman: false,
	}

	// The server unmarshals and re-marshals the payload, so we just match the query structure
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO nodes")).
		WithArgs("test-server-1", NodeType("Concept"), sqlmock.AnyArg(), 0.95, false, nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	res, err := server.Memorize(context.Background(), req)
	if err != nil {
		t.Errorf("Server Memorize failed: %v", err)
	}
	if res.Id != "test-server-1" {
		t.Errorf("Expected ID 'test-server-1', got '%s'", res.Id)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestServer_Recall(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	engine := &PostgresEngine{db: db}
	server := NewServer(engine)

	now := time.Now()
	payloadJSON := `{"data": "content"}`

	// 1. Mock GetNode
	rowsNode := sqlmock.NewRows([]string{"id", "type", "payload", "confidence", "requires_human", "ttl", "created_at", "updated_at"}).
		AddRow("center-node-1", "Fact", []byte(payloadJSON), 1.0, false, nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at FROM nodes WHERE id = $1")).
		WithArgs("center-node-1").
		WillReturnRows(rowsNode)

	// 2. Mock Outgoing Edges
	rowsOut := sqlmock.NewRows([]string{"source_id", "target_id", "relation", "created_at"}).
		AddRow("center-node-1", "target-node-2", "THEORY_OF", now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT source_id, target_id, relation, created_at FROM edges WHERE source_id = $1")).
		WithArgs("center-node-1").
		WillReturnRows(rowsOut)

	// 3. Mock Incoming Edges
	rowsIn := sqlmock.NewRows([]string{"source_id", "target_id", "relation", "created_at"})
	// Return empty for incoming edges

	mock.ExpectQuery(regexp.QuoteMeta("SELECT source_id, target_id, relation, created_at FROM edges WHERE target_id = $1")).
		WithArgs("center-node-1").
		WillReturnRows(rowsIn)

	// 4. Mock Neighbor Nodes
	rowsNeighbors := sqlmock.NewRows([]string{"id", "type", "payload", "confidence", "requires_human", "ttl", "created_at", "updated_at"}).
		AddRow("target-node-2", "Fact", []byte(`{"data":"neighbor"}`), 1.0, false, nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at FROM nodes WHERE id = ANY($1)")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rowsNeighbors)

	req := &pb.RecallRequest{
		Id: "center-node-1",
	}

	res, err := server.Recall(context.Background(), req)
	if err != nil {
		t.Fatalf("Server Recall failed: %v", err)
	}

	if res.Center.Id != "center-node-1" {
		t.Errorf("Expected Center ID 'center-node-1', got '%s'", res.Center.Id)
	}
	if len(res.OutEdges) != 1 || res.OutEdges[0].TargetId != "target-node-2" {
		t.Errorf("Expected 1 outgoing edge to 'target-node-2', got: %v", res.OutEdges)
	}
	if len(res.Neighbors) != 1 || res.Neighbors[0].Id != "target-node-2" {
		t.Errorf("Expected 1 neighbor node 'target-node-2', got: %v", res.Neighbors)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
