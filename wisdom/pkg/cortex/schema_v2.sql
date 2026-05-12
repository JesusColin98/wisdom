-- Universal Graph Schema for Wisdom (TRACK 01: Cortex Substrate)

-- Enum types for nodes and relations
DO $$ BEGIN
    CREATE TYPE node_type AS ENUM ('Fact', 'Signal', 'Concept', 'User');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE relation_type AS ENUM ('THEORY_OF', 'CONTRADICTS', 'PREREQUISITE_OF', 'MASTERED_BY');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Table: nodes
CREATE TABLE IF NOT EXISTS nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type node_type NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    confidence FLOAT DEFAULT 1.0,
    requires_human BOOLEAN DEFAULT false,
    ttl TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Table: edges
CREATE TABLE IF NOT EXISTS edges (
    source_id UUID REFERENCES nodes(id) ON DELETE CASCADE,
    target_id UUID REFERENCES nodes(id) ON DELETE CASCADE,
    relation relation_type NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relation)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_nodes_payload_gin ON nodes USING GIN (payload);
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
