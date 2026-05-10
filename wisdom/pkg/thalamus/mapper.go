package thalamus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// MapperService automatically maps codebase relationships.
type MapperService struct {
	Cortex *cortex.Cortex
	Chat   *Chat
}

// NewMapperService initializes a new mapper service.
func NewMapperService(c *cortex.Cortex, chat *Chat) *MapperService {
	return &MapperService{Cortex: c, Chat: chat}
}

// MapDirectory analyzes files in a directory and populates the Cortex with symbol relationships.
func (s *MapperService) MapDirectory(ctx context.Context, root string) (int, error) {
	ctx, span := observability.Tracer.Start(ctx, "Mapper.MapDirectory")
	defer span.End()

	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}

	mappedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		path := filepath.Join(root, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// 1. Ask LLM to extract symbols and dependencies
		prompt := fmt.Sprintf(`Analyze the following source code and extract:
1. Provided Symbols (Functions, Classes, Structs).
2. Dependencies (Imported packages or other symbols used).

Format as:
Symbols: symbol1, symbol2
Dependencies: dep1, dep2

Code from file %s:
%s`, entry.Name(), string(data))

		analysis, err := s.Chat.LLM.Complete(ctx, prompt)
		if err != nil {
			continue
		}

		// 2. Create Node for the File
		fileNode := &cortex.Node{
			ID:          "file-" + entry.Name(),
			Content:     fmt.Sprintf("Source file: %s", entry.Name()),
			EntityClass: "FILE",
			Author:      "wisdom-mapper",
			SourceType:  "CODEBASE",
			SourceRef:   path,
			NamespaceID: "ns-engineering",
		}
		_ = s.Cortex.PutNode(ctx, fileNode)

		// 3. Parse analysis and create links
		lines := strings.Split(analysis, "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.ToLower(line), "symbols:") {
				symbols := strings.Split(line[8:], ",")
				for _, sym := range symbols {
					sym = strings.TrimSpace(sym)
					if sym == "" {
						continue
					}
					symNode := &cortex.Node{
						ID:          "symbol-" + sym,
						Content:     fmt.Sprintf("Symbol %s defined in %s", sym, entry.Name()),
						EntityClass: "SYMBOL",
						Author:      "wisdom-mapper",
						NamespaceID: "ns-engineering",
					}
					_ = s.Cortex.PutNode(ctx, symNode)
					_ = s.Cortex.LinkNodes(ctx, &cortex.Link{
						SourceID:     fileNode.ID,
						TargetID:     symNode.ID,
						RelationType: "DEFINES",
						Weight:       1.0,
					})
				}
			} else if strings.HasPrefix(strings.ToLower(line), "dependencies:") {
				deps := strings.Split(line[13:], ",")
				for _, dep := range deps {
					dep = strings.TrimSpace(dep)
					if dep == "" {
						continue
					}
					_ = s.Cortex.LinkNodes(ctx, &cortex.Link{
						SourceID:     fileNode.ID,
						TargetID:     "symbol-" + dep, // Optimistic link to symbol
						RelationType: "DEPENDS_ON",
						Weight:       0.8,
					})
				}
			}
		}
		mappedCount++
	}

	return mappedCount, nil
}
