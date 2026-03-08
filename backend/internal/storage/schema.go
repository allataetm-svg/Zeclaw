package storage

// SchemaSQL contains all DDL statements required to initialise (or migrate)
// the Zeclaw SQLite database.  It is idempotent: every CREATE TABLE and
// CREATE INDEX uses IF NOT EXISTS so it can be re-run safely on start-up.
const SchemaSQL = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    system_prompt TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('main', 'sub')),
    parent_id TEXT REFERENCES agents(id),
    tools TEXT NOT NULL DEFAULT '[]',
    llm_endpoint_id TEXT REFERENCES endpoints(id),
    status TEXT NOT NULL DEFAULT 'idle',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,
    tool_name TEXT,
    tool_input TEXT,
    is_interrupted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS endpoints (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('cloud', 'ollama')),
    url TEXT NOT NULL,
    api_key TEXT,
    model TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    content TEXT NOT NULL,
    category TEXT NOT NULL CHECK(category IN ('fact', 'preference', 'instruction', 'learned')),
    importance REAL NOT NULL DEFAULT 0.5,
    confidence REAL NOT NULL DEFAULT 1.0,
    last_accessed TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cron_tasks (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    expression TEXT,
    interval TEXT,
    run_at TIMESTAMP,
    task TEXT NOT NULL,
    run_mode TEXT NOT NULL DEFAULT 'main' CHECK(run_mode IN ('main', 'isolated')),
    announce BOOLEAN DEFAULT TRUE,
    one_shot BOOLEAN DEFAULT FALSE,
    enabled BOOLEAN DEFAULT TRUE,
    last_run TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS heartbeat_config (
    agent_id TEXT PRIMARY KEY REFERENCES agents(id),
    enabled BOOLEAN DEFAULT FALSE,
    interval_seconds INTEGER DEFAULT 1800,
    active_hours_start TEXT DEFAULT '08:00',
    active_hours_end TEXT DEFAULT '22:00',
    target TEXT DEFAULT 'chat' CHECK(target IN ('chat', 'silent')),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS hooks (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    event TEXT NOT NULL,
    action TEXT NOT NULL,
    config TEXT NOT NULL DEFAULT '{}',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES agents(id),
    session_id TEXT,
    event_type TEXT NOT NULL,
    data TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS approval_requests (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    tool TEXT NOT NULL,
    input TEXT NOT NULL,
    risk_level TEXT NOT NULL CHECK(risk_level IN ('low', 'medium', 'high', 'critical')),
    reason TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'denied', 'expired')),
    responded_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    name TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    tools_required TEXT DEFAULT '[]',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_agent ON messages(agent_id, created_at);
CREATE INDEX IF NOT EXISTS idx_agents_parent ON agents(parent_id);
CREATE INDEX IF NOT EXISTS idx_memories_agent ON memories(agent_id, category);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(agent_id, importance DESC);
CREATE INDEX IF NOT EXISTS idx_cron_agent ON cron_tasks(agent_id, enabled);
CREATE INDEX IF NOT EXISTS idx_audit_agent ON audit_log(agent_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_log(event_type, created_at);
CREATE INDEX IF NOT EXISTS idx_approval_status ON approval_requests(status, created_at);
CREATE INDEX IF NOT EXISTS idx_skills_agent ON skills(agent_id, enabled);
`
