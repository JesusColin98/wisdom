package mcp

import (
	"encoding/json"
	"testing"
)

func TestJSONRPCParsing(t *testing.T) {
	data := []byte(`{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}`)
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.Method != "tools/list" {
		t.Errorf("Expected method tools/list, got %s", req.Method)
	}
}
