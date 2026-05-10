package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/google/wisdom/pkg/observability"
	"github.com/google/wisdom/pkg/thalamus"
)

// Server implements a native Go MCP server over stdio.
type Server struct {
	orchestrator *thalamus.Orchestrator
	chat         *thalamus.Chat
	rem          *thalamus.REMService
	reader       *bufio.Reader
	writer       io.Writer
	mu           sync.Mutex
}

// NewServer creates a new MCP server.
func NewServer(orch *thalamus.Orchestrator, chat *thalamus.Chat, rem *thalamus.REMService) *Server {
	return &Server{
		orchestrator: orch,
		chat:         chat,
		rem:          rem,
		reader:       bufio.NewReader(os.Stdin),
		writer:       os.Stdout,
	}
}

// Listen starts the stdio JSON-RPC loop.
func (s *Server) Listen(ctx context.Context) error {
	observability.Logger.Info("Native Go MCP Server listening on stdio")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := s.reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			go s.handleRequest(ctx, line)
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, data []byte) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		s.sendError(nil, -32700, "Parse error", err.Error())
		return
	}

	var result interface{}
	var mcpErr *Error

	switch req.Method {
	case "initialize":
		result = InitializeResponse{
			ProtocolVersion: "2024-11-05",
			ServerInfo: ServerInfo{
				Name:    "Wisdom-Native-MCP",
				Version: "1.0.0",
			},
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		}

	case "tools/list":
		result = ToolListResponse{
			Tools: []Tool{
				{
					Name:        "recall_wisdom",
					Description: "Retrieves pattern-aware context from Wisdom knowledge base.",
					InputSchema: json.RawMessage(`{
						"type": "object",
						"properties": {
							"query": {"type": "string"},
							"user_id": {"type": "string"},
							"seeds": {"type": "array", "items": {"type": "string"}},
							"uncertainty": {"type": "number"}
						},
						"required": ["query"]
					}`),
				},
				{
					Name:        "calculate_risk",
					Description: "Analyzes system risk for a given entity.",
					InputSchema: json.RawMessage(`{
						"type": "object",
						"properties": {
							"node_id": {"type": "string"},
							"depth": {"type": "integer"}
						},
						"required": ["node_id"]
					}`),
				},
				{
					Name:        "chat",
					Description: "Sends a message to the Wisdom assistant and gets a grounded response.",
					InputSchema: json.RawMessage(`{
						"type": "object",
						"properties": {
							"message": {"type": "string"},
							"user_id": {"type": "string"}
						},
						"required": ["message"]
					}`),
				},
				{
					Name:        "rem",
					Description: "Triggers a Rapid Epistemic Metabolism (REM) cycle to consolidate session nodes.",
					InputSchema: json.RawMessage(`{
						"type": "object",
						"properties": {
							"session_id": {"type": "string"}
						}
					}`),
				},
			},
		}

	case "tools/call":
		var callParams CallToolRequest
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			mcpErr = &Error{Code: -32602, Message: "Invalid params"}
		} else {
			res, err := s.dispatchTool(ctx, callParams.Name, callParams.Arguments)
			if err != nil {
				result = CallToolResponse{
					Content: []Content{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
					IsError: true,
				}
			} else {
				result = res
			}
		}

	case "notifications/initialized":
		return // Ignore

	default:
		// Legacy bridge support (non-standard MCP methods)
		switch req.Method {
		case "chat":
			var p struct { Message string `json:"message"` }
			json.Unmarshal(req.Params, &p)
			resp, _, _ := s.chat.Ask(ctx, "anonymous", p.Message)
			result = map[string]string{"response": resp}
		case "rem":
			var p struct { SessionID string `json:"session_id"` }
			json.Unmarshal(req.Params, &p)
			count, _ := s.rem.ConsolidateSession(ctx, p.SessionID)
			result = map[string]int{"anchored_nodes": count}
		default:
			mcpErr = &Error{Code: -32601, Message: "Method not found"}
		}
	}

	s.sendResponse(req.ID, result, mcpErr)
}

func (s *Server) dispatchTool(ctx context.Context, name string, args json.RawMessage) (*CallToolResponse, error) {
	switch name {
	case "recall_wisdom":
		var p struct {
			Query       string   `json:"query"`
			UserID      string   `json:"user_id"`
			Seeds       []string `json:"seeds"`
			Uncertainty float64  `json:"uncertainty"`
		}
		json.Unmarshal(args, &p)
		if p.UserID == "" { p.UserID = "anonymous" }
		
		cognition, err := s.orchestrator.Recall(ctx, p.UserID, p.Query, p.Seeds, 0, p.Uncertainty)
		if err != nil {
			return nil, err
		}

		out, _ := json.Marshal(cognition.Wisdom)
		return &CallToolResponse{
			Content: []Content{{Type: "text", Text: string(out)}},
		}, nil

	case "calculate_risk":
		var p struct {
			NodeID string `json:"node_id"`
			Depth  int    `json:"depth"`
		}
		json.Unmarshal(args, &p)
		if p.Depth == 0 { p.Depth = 2 }

		risks, err := s.orchestrator.CalculateRisk(ctx, p.NodeID, p.Depth)
		if err != nil {
			return nil, err
		}

		out, _ := json.Marshal(risks)
		return &CallToolResponse{
			Content: []Content{{Type: "text", Text: string(out)}},
		}, nil

	case "chat":
		var p struct {
			Message string `json:"message"`
			UserID  string `json:"user_id"`
		}
		json.Unmarshal(args, &p)
		if p.UserID == "" { p.UserID = "anonymous" }

		resp, nodes, err := s.chat.Ask(ctx, p.UserID, p.Message)
		if err != nil {
			return nil, err
		}

		res := map[string]interface{}{
			"response": resp,
			"context_nodes": nodes,
		}
		out, _ := json.Marshal(res)
		return &CallToolResponse{
			Content: []Content{{Type: "text", Text: string(out)}},
		}, nil

	case "rem":
		var p struct {
			SessionID string `json:"session_id"`
		}
		json.Unmarshal(args, &p)
		if p.SessionID == "" { p.SessionID = "anonymous" }

		count, err := s.rem.ConsolidateSession(ctx, p.SessionID)
		if err != nil {
			return nil, err
		}

		return &CallToolResponse{
			Content: []Content{{Type: "text", Text: fmt.Sprintf("Consolidated %d nodes.", count)}},
		}, nil

	default:
		return nil, fmt.Errorf("tool not implemented: %s", name)
	}
}

func (s *Server) sendResponse(id interface{}, result interface{}, err *Error) {
	resp := Response{
		JSONRPC: Version,
		ID:      id,
		Result:  result,
		Error:   err,
	}

	data, _ := json.Marshal(resp)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writer.Write(data)
	s.writer.Write([]byte("\n"))
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	s.sendResponse(id, nil, &Error{Code: code, Message: message, Data: data})
}
