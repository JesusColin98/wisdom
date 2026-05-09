-- Wisdom Cortex Schema (SQLite)
-- Purpose: High-performance semantic memory with strict provenance.

-- Namespaces for logical isolation of engineering domains
CREATE TABLE IF NOT EXISTS namespaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Wisdom Nodes: The core units of information
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,               -- UUID or Human-friendly Alias
    content TEXT NOT NULL,             -- The actual fact or observation
    entity_class TEXT NOT NULL DEFAULT 'OBSERVATION', -- PERSON, ROLE, CONCEPT, ERROR_PATTERN, PROCEDURE, INCIDENT
    author TEXT NOT NULL,              -- LDAP of the user (e.g., jesuscolin)
    source_type TEXT NOT NULL,         -- BUGANIZER, TABLE, URL, MANUAL, G3DOC, REM_CYCLE
    source_ref TEXT,                   -- b/123, table_name, https://..., etc.
    namespace_id TEXT NOT NULL,
    metadata JSON DEFAULT '{}',        -- Typed attributes (e.g., owner, dashboards)
    confidence_score REAL DEFAULT 0.8, -- 0.0 - 1.0 (The Truth Metric)
    superseded_by_id TEXT,             -- Traceable Neurogenesis: Link to newer version
    valid_from TIMESTAMP,              -- Temporal logic start
    valid_until TIMESTAMP,             -- Temporal logic end
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (namespace_id) REFERENCES namespaces(id),
    FOREIGN KEY (superseded_by_id) REFERENCES nodes(id)
);

-- Indices for performance
CREATE INDEX IF NOT EXISTS idx_nodes_author ON nodes(author);
CREATE INDEX IF NOT EXISTS idx_nodes_entity_class ON nodes(entity_class);
CREATE INDEX IF NOT EXISTS idx_nodes_source ON nodes(source_type, source_ref);
CREATE INDEX IF NOT EXISTS idx_nodes_namespace ON nodes(namespace_id);

-- Links: Directed relationships between nodes with rich semantics
CREATE TABLE IF NOT EXISTS links (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relation_type TEXT NOT NULL,       -- PARENT_OF, DEPENDS_ON, REMEDIATED_BY, SYNONYM_OF, CONFLICTS_WITH, PRECEDES
    weight REAL DEFAULT 1.0,           -- Strength of relationship
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relation_type),
    FOREIGN KEY (source_id) REFERENCES nodes(id),
    FOREIGN KEY (target_id) REFERENCES nodes(id)
);

CREATE INDEX IF NOT EXISTS idx_links_target ON links(target_id);

-- Vectors: Semantic embeddings for nodes
CREATE TABLE IF NOT EXISTS vectors (
    node_id TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,           -- Float32 array serialized as bytes
    model_version TEXT NOT NULL,       -- e.g., "minilm-v2"
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Node History: Archival of previous node versions
CREATE TABLE IF NOT EXISTS node_history (
    history_id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata JSON,
    version_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (node_id) REFERENCES nodes(id)
);

-- Trigger to automatically archive versions on update
CREATE TRIGGER IF NOT EXISTS archive_node_version
BEFORE UPDATE ON nodes
BEGIN
    INSERT INTO node_history (node_id, content, metadata)
    VALUES (OLD.id, OLD.content, OLD.metadata);
END;

-- Session Logs: Persistent Hippocampus for transient interactions
CREATE TABLE IF NOT EXISTS session_logs (
    log_id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,           -- user, assistant, tool, system
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_session_logs_id ON session_logs(session_id);

-- Tools: Dynamic tool definitions (Neurogenesis)
CREATE TABLE IF NOT EXISTS tools (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    source_code TEXT NOT NULL,        -- Go source for Yaegi
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
