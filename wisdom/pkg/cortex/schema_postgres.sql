-- Project Wisdom: PostgreSQL Schema (optimized for pgvector and multi-hop CTEs)
-- To be used with Neon/Supabase for the Low-Cost Serverless Tier.

CREATE EXTENSION IF NOT EXISTS vector;

-- Nodes Table: Core semantic entities
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    entity_class TEXT,
    author TEXT,
    source_type TEXT,
    source_ref TEXT,
    namespace_id TEXT,
    metadata JSONB DEFAULT '{}',
    confidence_score DOUBLE PRECISION DEFAULT 1.0,
    impact_score DOUBLE PRECISION DEFAULT 0.0,
    stratum TEXT DEFAULT 'HOT', -- 'HOT' or 'COLD'
    source_mime_type TEXT DEFAULT 'text/plain',
    external_links JSONB DEFAULT '[]',
    superseded_by_id TEXT REFERENCES nodes(id),
    valid_from TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP WITH TIME ZONE,
    repetition_count INTEGER DEFAULT 0,
    easiness_factor DOUBLE PRECISION DEFAULT 2.5,
    next_review_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Vectors Table: High-dimensional semantic embeddings
CREATE TABLE IF NOT EXISTS vectors (
    node_id TEXT PRIMARY KEY REFERENCES nodes(id) ON DELETE CASCADE,
    embedding vector(768), -- Standardized 768-dim (e.g. text-embedding-004)
    model_version TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Links Table: Explicit graph edges
CREATE TABLE IF NOT EXISTS links (
    source_id TEXT REFERENCES nodes(id) ON DELETE CASCADE,
    target_id TEXT REFERENCES nodes(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL,
    weight DOUBLE PRECISION DEFAULT 1.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relation_type)
);

-- Session Logs: Conversational history
CREATE TABLE IF NOT EXISTS session_logs (
    log_id SERIAL PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tools Table: Dynamic cerebellum definitions
CREATE TABLE IF NOT EXISTS tools (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    source_code TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_nodes_namespace ON nodes(namespace_id);
CREATE INDEX IF NOT EXISTS idx_nodes_stratum ON nodes(stratum);
CREATE INDEX IF NOT EXISTS idx_nodes_entity_class ON nodes(entity_class);
CREATE INDEX IF NOT EXISTS idx_session_logs_session_id ON session_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_links_target ON links(target_id);

-- pgvector HNSW Index for millisecond semantic search
CREATE INDEX IF NOT EXISTS idx_vectors_hnsw ON vectors USING hnsw (embedding vector_cosine_ops);
