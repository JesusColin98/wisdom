package cortex

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresEngine_Memorize(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	engine := &PostgresEngine{db: db}

	node := &Node{
		ID:         "test-uuid-1",
		Type:       NodeSignal,
		Payload:    map[string]any{"key": "value"},
		Confidence: 0.9,
	}

	payloadJSON, _ := json.Marshal(node.Payload)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO nodes")).
		WithArgs(node.ID, node.Type, payloadJSON, node.Confidence, node.RequiresHuman, node.TTL).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = engine.Memorize(context.Background(), node)
	if err != nil {
		t.Errorf("Memorize failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestPostgresEngine_GetNode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	engine := &PostgresEngine{db: db}

	payloadJSON := `{"key": "value"}`
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "type", "payload", "confidence", "requires_human", "ttl", "created_at", "updated_at"}).
		AddRow("test-uuid-1", "Fact", []byte(payloadJSON), 1.0, false, nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at FROM nodes WHERE id = $1")).
		WithArgs("test-uuid-1").
		WillReturnRows(rows)

	node, err := engine.GetNode(context.Background(), "test-uuid-1")
	if err != nil {
		t.Errorf("GetNode failed: %v", err)
	}
	if node == nil || node.ID != "test-uuid-1" || node.Type != NodeFact {
		t.Errorf("Unexpected node returned: %v", node)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestPostgresEngine_QueryHechos(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	engine := &PostgresEngine{db: db}

	filters := map[string]string{"topic": "chess"}
	filterJSON, _ := json.Marshal(filters)
	payloadJSON := `{"topic": "chess"}`
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "type", "payload", "confidence", "requires_human", "ttl", "created_at", "updated_at"}).
		AddRow("test-uuid-2", "Fact", []byte(payloadJSON), 1.0, false, nil, now, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, payload, confidence, requires_human, ttl, created_at, updated_at FROM nodes WHERE type = 'Fact' AND payload @> $1")).
		WithArgs(filterJSON).
		WillReturnRows(rows)

	facts, err := engine.QueryHechos(context.Background(), filters)
	if err != nil {
		t.Errorf("QueryHechos failed: %v", err)
	}
	if len(facts) != 1 || facts[0].ID != "test-uuid-2" {
		t.Errorf("Unexpected facts returned: %v", facts)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
