package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/metabolism"
	"github.com/google/wisdom/pkg/sensory"
	"github.com/google/wisdom/pkg/thalamus"
)

func TestServerEndpoints(t *testing.T) {
	ctx := context.Background()

	// Initialize dependencies
	testDB := "test_api.db"
	os.Remove(testDB)
	defer os.Remove(testDB)
	defer os.Remove(testDB + ".rpforest")

	storage, err := cortex.Open(testDB)
	if err != nil {
		t.Fatalf("failed to open cortex: %v", err)
	}
	defer storage.Close()

	// Load schema
	schemaSQL := `
		CREATE TABLE namespaces (id TEXT PRIMARY KEY, name TEXT, description TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE nodes (
		        id TEXT PRIMARY KEY, 
		        content TEXT NOT NULL, 
		        entity_class TEXT NOT NULL DEFAULT 'OBSERVATION',
		        author TEXT NOT NULL, 
		        source_type TEXT NOT NULL, 
		        source_ref TEXT, 
		        namespace_id TEXT, 
		        metadata JSON, 
		        confidence_score REAL DEFAULT 0.8,
		        impact_score REAL DEFAULT 0.0,
		        stratum TEXT NOT NULL DEFAULT 'HOT',
		        source_mime_type TEXT DEFAULT 'text/plain',
		        external_links JSON DEFAULT '[]',
		        created_at DATETIME DEFAULT CURRENT_TIMESTAMP, 
		        updated_at DATETIME, 
		        FOREIGN KEY(namespace_id) REFERENCES namespaces(id)
		);
		CREATE TABLE links (source_id TEXT, target_id TEXT, relation_type TEXT, weight REAL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY(source_id, target_id, relation_type), FOREIGN KEY(source_id) REFERENCES nodes(id) ON DELETE CASCADE, FOREIGN KEY(target_id) REFERENCES nodes(id) ON DELETE CASCADE);
		CREATE TABLE vectors (node_id TEXT PRIMARY KEY, embedding BLOB, model_version TEXT, updated_at DATETIME, FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE);
		CREATE TABLE session_logs (log_id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT, role TEXT, content TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);

		CREATE TABLE node_history (history_id INTEGER PRIMARY KEY AUTOINCREMENT, node_id TEXT, content TEXT, metadata JSON, version_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);
	`
	if err := storage.InitSchema(ctx, schemaSQL); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	tracker := metabolism.NewTracker()
	validator := thalamus.NewValidator()
	registry := cerebellum.NewRegistry()
	cache, _ := thalamus.NewCache(10)
	hippocampus := thalamus.NewHippocampus(storage)

	chat := &thalamus.Chat{
		Storage:     storage,
		LLM:         &cerebellum.MockLLM{CannedResponse: "Chat response"},
		Hippocampus: hippocampus,
	}
	remService := &thalamus.REMService{
		Hippocampus: hippocampus,
		Cortex:      storage,
		Chat:        chat,
	}
	orchestrator := thalamus.NewOrchestrator(storage, cache, nil, nil, nil, nil, nil, nil, nil, nil)
	chat.Orchestrator = orchestrator

	scheduler := &thalamus.Scheduler{}
	ingestor := &sensory.DocumentIngestor{}
	mapper := &thalamus.MapperService{}
	hierarchy := &thalamus.HierarchyManager{}

	server := NewServer(storage, tracker, validator, registry, chat, remService, orchestrator, scheduler, ingestor, mapper, hierarchy)
	mux := http.NewServeMux()
	server.RegisterHandlers(mux)

	t.Run("GET /health", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /reason", func(t *testing.T) {
		reqBody := `{"query": "logic"}`
		req := httptest.NewRequest("POST", "/reason", bytes.NewBufferString(reqBody))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /validate", func(t *testing.T) {
		reqBody := `{"assertion": "test"}`
		req := httptest.NewRequest("POST", "/validate", bytes.NewBufferString(reqBody))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("POST /cortex/upload", func(t *testing.T) {
		// Prepare a multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("document", "test.txt")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("this is some test content for the document"))
		writer.Close()

		// Set up ingestor correctly for the test
		server.ingestor = sensory.NewDocumentIngestor(&cerebellum.MockLLM{}, storage)

		req := httptest.NewRequest("POST", "/cortex/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})
}
