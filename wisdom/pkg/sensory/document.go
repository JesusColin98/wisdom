package sensory

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/wisdom/pkg/cerebellum"
	"github.com/google/wisdom/pkg/cortex"
	"github.com/google/wisdom/pkg/observability"
)

// DocumentIngestor extracts knowledge from documents using multimodal LLMs.
type DocumentIngestor struct {
	LLM    cerebellum.LLMProvider
	Cortex *cortex.Cortex
}

// NewDocumentIngestor creates a new document ingestor.
func NewDocumentIngestor(llm cerebellum.LLMProvider, storage *cortex.Cortex) *DocumentIngestor {
	return &DocumentIngestor{
		LLM:    llm,
		Cortex: storage,
	}
}

// Ingest parses a document and anchors findings into the Cortex.
func (i *DocumentIngestor) Ingest(ctx context.Context, data []byte, mimeType, sourceRef, author string) (int, error) {
	ctx, span := observability.Tracer.Start(ctx, "DocumentIngestor.Ingest")
	defer span.End()

	var wisdom string
	var err error

	// 1. Handle based on type
	if strings.HasPrefix(mimeType, "text/") {
		// Plain text: Send as direct prompt context
		prompt := fmt.Sprintf("Extract all key engineering findings, entities, and relationships from the following text data:\n\n%s", string(data))
		wisdom, err = i.LLM.Complete(ctx, prompt)
	} else {
		// Binary (PDF, Image, Office, etc.): Use Multimodal interface
		wisdom, err = i.LLM.IngestDocument(ctx, data, mimeType)
	}

	if err != nil {
		return 0, fmt.Errorf("extraction failed for %s (%s): %w", sourceRef, mimeType, err)
	}

	// 2. Anchor findings to Cortex
	node := &cortex.Node{
		ID:              fmt.Sprintf("doc-%s", sourceRef),
		Content:         wisdom,
		Author:          author,
		SourceType:      "DOCUMENT",
		SourceRef:       sourceRef,
		NamespaceID:     "ns-general",
		ConfidenceScore: 0.7,
		Metadata: map[string]any{
			"mime_type": mimeType,
			"size":      len(data),
		},
	}

	if err := i.Cortex.PutNode(ctx, node); err != nil {
		return 0, err
	}

	// 3. Generate embedding for search
	embedding, err := i.LLM.Embed(ctx, wisdom)
	if err == nil {
		_ = i.Cortex.PutVector(ctx, node.ID, embedding, "multimodal-v1")
	}

	return 1, nil
}
