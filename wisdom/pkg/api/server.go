package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/metabolism"
	"github.com/google/wisdom/pkg/observability"
	"github.com/google/wisdom/pkg/sensory"
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
	scheduler    *thalamus.Scheduler
	ingestor     *sensory.DocumentIngestor
	mapper       *thalamus.MapperService
	hierarchy    *thalamus.HierarchyManager
	wsManager    *WSManager
}

// NewServer initializes a new API server with its dependencies.
func NewServer(
	storage *cortex.Cortex,
	tracker *metabolism.Tracker,
	validator *thalamus.Validator,
	registry *cerebellum.Registry,
	chat *thalamus.Chat,
	rem *thalamus.REMService,
	orchestrator *thalamus.Orchestrator,
	scheduler *thalamus.Scheduler,
	ingestor *sensory.DocumentIngestor,
	mapper *thalamus.MapperService,
	hierarchy *thalamus.HierarchyManager,
) *Server {
	return &Server{
		storage:      storage,
		tracker:      tracker,
		validator:    validator,
		registry:     registry,
		chat:         chat,
		rem:          rem,
		orchestrator: orchestrator,
		scheduler:    scheduler,
		ingestor:     ingestor,
		mapper:       mapper,
		hierarchy:    hierarchy,
		wsManager:    NewWSManager(),
	}
}

// RegisterHandlers registers the API endpoints with the provided mux.
func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /metabolism", s.handleMetabolism)
	mux.HandleFunc("GET /cortex/nodes", s.handleListNodes)
	mux.HandleFunc("GET /cortex/edges", s.handleListEdges)
	mux.HandleFunc("GET /cortex/due", s.handleGetDueNodes)
	mux.HandleFunc("POST /cortex/review", s.handleReviewNode)
	mux.HandleFunc("POST /cortex/upload", s.handleUploadDocument)
	mux.HandleFunc("POST /cortex/map", s.handleMapCodebase)
	mux.HandleFunc("POST /chat", s.handleChat)
	mux.HandleFunc("POST /cortex/notes", s.handleCreateNote)
	mux.HandleFunc("POST /rem", s.handleREM)
	mux.HandleFunc("POST /rem/all", s.handleREMAll)
	mux.HandleFunc("POST /reason", s.handleReason)
	mux.HandleFunc("POST /validate", s.handleValidate)
	mux.HandleFunc("GET /cortex/impact", s.handleImpact)
	mux.HandleFunc("GET /cortex/lineage", s.handleLineage)
	mux.HandleFunc("GET /cortex/risk", s.handleRisk)
	mux.HandleFunc("GET /cortex/causality", s.handleCausality)
	mux.HandleFunc("POST /cortex/recall", s.handleRecall)
	mux.HandleFunc("POST /cortex/upvote", s.handleUpvote)
	mux.HandleFunc("GET /ws", s.handleWS)
	mux.HandleFunc("POST /config", s.handleUpdateConfig)

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

func (s *Server) handleGetDueNodes(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleGetDueNodes")
	defer span.End()

	namespace := r.URL.Query().Get("namespace")
	limit := 10
	nodes, err := s.scheduler.GetDueNodes(ctx, namespace, limit)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleReviewNode(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleReviewNode")
	defer span.End()

	var req struct {
		NodeID string `json:"node_id"`
		Grade  int    `json:"grade"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := s.scheduler.ReviewNode(ctx, req.NodeID, req.Grade); err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]string{"status": "REVIEWED", "id": req.NodeID})
}

func (s *Server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleUploadDocument")
	defer span.End()

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large"})
		return
	}

	file, header, err := r.FormFile("document")
	if err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "missing document"})
		return
	}
	defer file.Close()

	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read file"})
		return
	}

	count, err := s.ingestor.Ingest(ctx, data, header.Header.Get("Content-Type"), header.Filename, "anonymous")
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]any{
		"status": "INGESTED",
		"nodes":  count,
	})
}

func (s *Server) handleMapCodebase(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleMapCodebase")
	defer span.End()

	var req struct {
		Directory string `json:"directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Directory == "" {
		req.Directory = "."
	}

	count, err := s.mapper.MapDirectory(ctx, req.Directory)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]any{
		"status": "MAPPED",
		"files":  count,
	})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleChat")
	defer span.End()

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	response, contextStrings, err := s.chat.Ask(ctx, "anonymous", req.Message)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Record metabolic usage (Heuristic for now)
	tokensIn := len(req.Message) / 4
	tokensOut := len(response) / 4
	s.tracker.Record("anonymous", metabolism.Usage{
		TokensIn:    tokensIn,
		TokensOut:   tokensOut,
		SignalUnits: len(contextStrings), // Nodes used as signal
		Duration:    time.Since(startTime),
	})
	s.broadcastMetabolism("anonymous")

	s.sendJSON(w, http.StatusOK, map[string]any{
		"response":      response,
		"context_nodes": contextStrings,
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

func (s *Server) handleLineage(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("id")
	direction := r.URL.Query().Get("direction") // UP or DOWN
	if nodeID == "" {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	nodes, err := s.hierarchy.GetLineage(r.Context(), nodeID, direction)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleRisk(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleRisk")
	defer span.End()

	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required"})
		return
	}

	risks, err := s.orchestrator.CalculateRisk(ctx, nodeID, 2)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, risks)
}

func (s *Server) handleCausality(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleCausality")
	defer span.End()

	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required"})
		return
	}

	chains, err := s.orchestrator.TraceCausality(ctx, nodeID)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, chains)
}

func (s *Server) handleRecall(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.Tracer.Start(r.Context(), "api.handleRecall")
	defer span.End()

	var req struct {
		Query       string   `json:"query"`
		UserID      string   `json:"user_id"`
		Seeds       []string `json:"seeds"`
		Uncertainty float64  `json:"uncertainty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.UserID == "" {
		req.UserID = "anonymous"
	}

	cognition, err := s.orchestrator.Recall(ctx, req.UserID, req.Query, req.Seeds, 0, req.Uncertainty)
	if err != nil {
		s.sendJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]any{
		"wisdom": cognition.Wisdom,
	})
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

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config thalamus.WisdomConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.sendJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid config body"})
		return
	}

	s.orchestrator.UpdateConfig(config)
	s.sendJSON(w, http.StatusOK, map[string]string{"status": "CONFIG_UPDATED"})
}

func (s *Server) broadcastMetabolism(sessionID string) {
	report := s.tracker.Efficiency(sessionID)
	s.wsManager.Broadcast(WSMessage{
		Type: "METABOLIC_UPDATE",
		Payload: map[string]any{
			"session_id": sessionID,
			"tsr":        report.TSR,
			"health":     report.HealthStatus,
			"tokens":     report.TotalTokens,
		},
	})
}

func (s *Server) sendJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}
