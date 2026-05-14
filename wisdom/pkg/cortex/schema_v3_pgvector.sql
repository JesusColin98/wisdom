-- Schema Migration V3: pgvector Semantic Search
-- Run after schema_v2.sql to add vector embeddings to Cortex nodes.
--
-- Prerequisites:
--   CREATE EXTENSION IF NOT EXISTS vector;  -- pgvector extension
--
-- This migration adds:
--   1. embedding column (vector(768)) on nodes — Vertex AI text-embedding-004 output
--   2. HNSW index for fast approximate nearest-neighbor search
--   3. Full-text search column (tsvector) for hybrid keyword+vector search
--   4. node_type extended with new domain types

-- 1. Enable pgvector extension.
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. Extend node_type enum with domain types introduced by the MoE refactoring.
DO $$ BEGIN
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'Opening';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'InvestmentThesis';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'PortfolioPosition';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'Vocabulary';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'GrammarRule';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'Algorithm';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'ADR';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'Theory';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'Procedure';
    ALTER TYPE node_type ADD VALUE IF NOT EXISTS 'Example';
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 3. Add embedding column (nullable — not all nodes have embeddings yet).
ALTER TABLE nodes
    ADD COLUMN IF NOT EXISTS embedding vector(768),
    ADD COLUMN IF NOT EXISTS embedding_model TEXT DEFAULT 'text-embedding-004',
    ADD COLUMN IF NOT EXISTS ts_content TSVECTOR; -- For hybrid full-text search.

-- 4. HNSW index for fast ANN search (cosine distance).
--    m=16, ef_construction=64 — balanced quality/build speed for <1M rows.
CREATE INDEX IF NOT EXISTS idx_nodes_embedding_hnsw
    ON nodes
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- 5. GIN index for full-text search on ts_content.
CREATE INDEX IF NOT EXISTS idx_nodes_ts_content
    ON nodes
    USING GIN (ts_content);

-- 6. Trigger to auto-populate ts_content from payload text fields.
CREATE OR REPLACE FUNCTION update_node_ts_content()
RETURNS TRIGGER AS $$
BEGIN
    NEW.ts_content = to_tsvector('english',
        COALESCE(NEW.payload->>'title', '') || ' ' ||
        COALESCE(NEW.payload->>'content', '') || ' ' ||
        COALESCE(NEW.payload->>'name', '') || ' ' ||
        COALESCE(NEW.payload->>'domain', '')
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_node_ts_content ON nodes;
CREATE TRIGGER trg_node_ts_content
    BEFORE INSERT OR UPDATE ON nodes
    FOR EACH ROW EXECUTE FUNCTION update_node_ts_content();

-- 7. Backfill ts_content for existing rows.
UPDATE nodes SET ts_content = to_tsvector('english',
    COALESCE(payload->>'title', '') || ' ' ||
    COALESCE(payload->>'content', '') || ' ' ||
    COALESCE(payload->>'name', '') || ' ' ||
    COALESCE(payload->>'domain', '')
) WHERE ts_content IS NULL;

-- 8. Add mastery_score to Signal nodes for Metabolism SRS.
ALTER TABLE nodes
    ADD COLUMN IF NOT EXISTS mastery_score FLOAT DEFAULT 0.5,
    ADD COLUMN IF NOT EXISTS next_review_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS review_count INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_nodes_next_review
    ON nodes (next_review_at)
    WHERE next_review_at IS NOT NULL;

-- Verification query (run after migration).
-- SELECT COUNT(*) FROM nodes WHERE embedding IS NOT NULL;
