package cerebellum

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/google/wisdom/pkg/observability"
)

// GrepRAGAgent provides sub-second lexical retrieval for codebases without indexing.
type GrepRAGAgent struct {
	RootPath string // Base directory for search
}

// NewGrepRAGAgent creates a new GrepRAGAgent.
func NewGrepRAGAgent(root string) *GrepRAGAgent {
	return &GrepRAGAgent{RootPath: root}
}

// Search retrieves code snippets matching the pattern using ripgrep (rg).
func (a *GrepRAGAgent) Search(ctx context.Context, pattern string, filePattern string) (string, error) {
	ctx, span := observability.Tracer.Start(ctx, "Cerebellum.GrepRAG.Search")
	defer span.End()

	// 1. Build rg command
	// -C 2: include 2 lines of context
	// --max-count 10: limit results to prevent context blowout
	// --heading: group results by file
	args := []string{"-C", "2", "--max-count", "10", "--heading", pattern}
	if filePattern != "" {
		args = append([]string{"-g", filePattern}, args...)
	}

	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = a.RootPath

	// 2. Execute
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Exit code 1 means no matches, which is not a failure for us
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No lexical matches found in codebase.", nil
		}
		return "", fmt.Errorf("ripgrep failed: %w (Output: %s)", err, string(output))
	}

	return string(output), nil
}

// MapSymbol performs a precise search for symbol definitions (Classes/Functions).
func (a *GrepRAGAgent) MapSymbol(ctx context.Context, symbol string) (string, error) {
	// Look for common definition patterns: 'func Symbol', 'class Symbol', 'type Symbol'
	pattern := fmt.Sprintf(`(func|class|type|struct|interface)\s+%s`, symbol)
	return a.Search(ctx, pattern, "")
}
