package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/metabolism"
	"github.com/google/wisdom/pkg/observability"
	"github.com/google/wisdom/pkg/thalamus"
)

// Server represents the high-performance HTTP API server for Project Wisdom.
type Server struct {
	storage      *cortex.Cortex
	tracker      *metabolism.Tracker
	validator    *thalamus.Validator
	registry     *cerebellum.Registry
	chat         *thalamus.Chat
	rem          *thalamus.REMService
	orchestrator *thalamus.Orchestrator
	wsManager    *WSManager
}

// NewServer initializes a new API server with its dependencies.
func NewServer(storage *cortex.Cortex, tracker *metabolism.Tracker, validator *thalamus.Validator, registry *cerebellum.Registry, chat *thalamus.Chat, rem *thalamus.REMService, orchestrator *thalamus.Orchestrator) *Server {
	return &Server{
		storage:      storage,
		tracker:      tracker,
		validator:    validator,
		registry:     registry,
		chat:         chat,
		rem:          rem,
		orchestrator: orchestrator,
		wsManager:    NewWSManager(),
	}
}

// RegisterHandlers registers the API endpoints with the provided mux.
func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /metabolism", s.handleMetabolism)
	mux.HandleFunc("GET /cortex/nodes", s.handleListNodes)
	mux.HandleFunc("GET /cortex/edges", s.handleListEdges)
	mux.HandleFunc("POST /chat", s.handleChat)
	mux.HandleFunc("POST /cortex/notes", s.handleCreateNote)
	mux.HandleFunc("POST /rem", s.handleREM)
	mux.HandleFunc("POST /rem/all", s.handleREMAll)
	mux.HandleFunc("POST /reason", s.handleReason)
	mux.HandleFunc("POST /validate", s.handleValidate)
	mux.HandleFunc("GET /cortex/impact", s.handleImpact)
	mux.HandleFunc("POST /cortex/upvote", s.handleUpvote)
	mux.HandleFunc("GET /ws", s.handleWS)

	// Static Frontend serving
	fs := http.FileServer(http.Dir("./public"))
	mux.Handle("GET /ui/", http.StripPrefix("/ui/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observability.Logger.Info("Serving UI asset", "path", r.URL.Path)
		fs.ServeHTTP(w, r)
	})))
	mux.Handle("GET /assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observability.Logger.Info("Serving static asset", "path", r.URL.Path)
		fs.ServeHTTP(w, r)
	}))

	// Redirect root to /ui/
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
			return
		}
		http.NotFound(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	_, span := observability.Tracer.Start(r.Context(), "api.handleHealth")
	defer span.End()

	s.sendJSON(w, http.StatusOK, map[string]string{"status": "OK"})
}

func (s *Server) handleMetabolism(w http.ResponseWriter, r *http.Request) {
	_, span := observability.Tracer.Start(r.Context(), "api.handleMetabolism")
	defer span.End()

	efficiency := s.tracker.GlobalEfficiency()
	s.sendJSON(w, http.StatusOK, efficiency)
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleListNodes")
	defer span.End()

	namespace := r.URL.Query().Get("namespace")
	nodes, err := s.storage.ListNodes(ctx, namespace)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleListEdges(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleListEdges")
	defer span.End()

	edges, err := s.storage.ListEdges(ctx)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, edges)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleChat")
	defer span.End()

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	response, nodes, err := s.chat.Ask(ctx, "anonymous", req.Message)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]any{
		"response":      response,
		"context_nodes": nodes,
	})
}

func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleCreateNote")
	defer span.End()

	var req struct {
		ID          string `json:"id"`
		Content     string `json:"content"`
		Author      string `json:"author"`
		NamespaceID string `json:"namespace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	node := cortex.Node{
		ID:          req.ID,
		Content:     req.Content,
		Author:      req.Author,
		NamespaceID: req.NamespaceID,
		SourceType:  "MANUAL",
	}

	if err := s.storage.PutNode(ctx, &node); err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{
		"status": "SAVED",
		"id":     node.ID,
	})
}

func (s *Server) handleREM(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleREM")
	defer span.End()

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = "anonymous"
	}

	count, err := s.rem.ConsolidateSession(ctx, sessionID)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Notify via WebSocket
	s.Notify("REM_CONSOLIDATED", map[string]any{
		"session_id":     sessionID,
		"anchored_nodes": count,
	})

	s.sendJSON(w, http.StatusOK, map[string]any{
		"status":         "CONSOLIDATED",
		"anchored_nodes": count,
	})
}

func (s *Server) handleREMAll(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleREMAll")
	defer span.End()

	count, err := s.rem.ConsolidateAllSessions(ctx, 1*time.Hour)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]any{
		"status":          "ALL_CONSOLIDATED",
		"total_anchored": count,
	})
}

func (s *Server) handleReason(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleReason")
	defer span.End()

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	explanation, nodes, err := s.chat.Reason(ctx, req.Query)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]any{
		"explanation": explanation,
		"nodes":       nodes,
	})
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleValidate")
	defer span.End()

	var req struct {
		Assertion string `json:"assertion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	isSafe, reason, err := s.chat.Validate(ctx, req.Assertion)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]any{
		"valid":  isSafe,
		"reason": reason,
	})
}

func (s *Server) handleImpact(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleImpact")
	defer span.End()

	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required"})
		return
	}

	nodes, err := s.orchestrator.GetImpactGraph(ctx, nodeID, 3)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleUpvote(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleUpvote")
	defer span.End()

	var req struct {
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Logic: Upvoting a node increases confidence by 0.1 (stronger than automated REM strengthening)
	if err := s.storage.UpdateConfidence(ctx, req.NodeID, 0.1); err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{"status": "CONFIDENCE_INCREASED", "id": req.NodeID})
}

func (s *Server) sendJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}
