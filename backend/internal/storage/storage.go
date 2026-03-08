package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

// Store wraps an SQLite database connection and exposes typed CRUD helpers.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at dbPath and runs all
// migrations defined in SchemaSQL.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrating db: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(SchemaSQL)
	return err
}

// Close shuts down the underlying database connection.
func (s *Store) Close() error { return s.db.Close() }

// ---------------------------------------------------------------------------
// Records
// ---------------------------------------------------------------------------

// AgentRecord mirrors a row in the agents table.
type AgentRecord struct {
	ID            string
	Name          string
	SystemPrompt  string
	Type          string
	ParentID      string
	Tools         []string
	LLMEndpointID string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// EndpointRecord mirrors a row in the endpoints table.
type EndpointRecord struct {
	ID        string
	Name      string
	Type      string
	URL       string
	APIKey    string
	Model     string
	IsDefault bool
	CreatedAt time.Time
}

// MessageRecord mirrors a row in the messages table.
type MessageRecord struct {
	ID            string
	AgentID       string
	Role          string
	Content       string
	ToolName      string
	ToolInput     string
	IsInterrupted bool
	CreatedAt     time.Time
}

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------

// CreateAgent inserts a new agent row and returns the fully-populated record.
func (s *Store) CreateAgent(name, systemPrompt, agentType, parentID, endpointID string, tools []string) (*AgentRecord, error) {
	id := uuid.New().String()
	toolsJSON, _ := json.Marshal(tools)
	_, err := s.db.Exec(
		`INSERT INTO agents (id, name, system_prompt, type, parent_id, tools, llm_endpoint_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, systemPrompt, agentType, nullStr(parentID), string(toolsJSON), nullStr(endpointID),
	)
	if err != nil {
		return nil, err
	}
	return s.GetAgent(id)
}

// GetAgent retrieves a single agent by ID.
func (s *Store) GetAgent(id string) (*AgentRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, name, system_prompt, type, COALESCE(parent_id,''), tools, COALESCE(llm_endpoint_id,''), status, created_at, updated_at FROM agents WHERE id = ?`,
		id,
	)
	return scanAgent(row)
}

// ListAgents returns all agents ordered by creation time.
func (s *Store) ListAgents() ([]*AgentRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, name, system_prompt, type, COALESCE(parent_id,''), tools, COALESCE(llm_endpoint_id,''), status, created_at, updated_at FROM agents ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agents []*AgentRecord
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// UpdateAgentStatus sets the status column for the given agent.
func (s *Store) UpdateAgentStatus(id, status string) error {
	_, err := s.db.Exec(
		`UPDATE agents SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, id,
	)
	return err
}

// DeleteAgent removes an agent (and cascades via FK where applicable).
func (s *Store) DeleteAgent(id string) error {
	_, err := s.db.Exec(`DELETE FROM agents WHERE id = ?`, id)
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanAgent(row scanner) (*AgentRecord, error) {
	var a AgentRecord
	var toolsJSON string
	var createdAt, updatedAt string
	err := row.Scan(
		&a.ID, &a.Name, &a.SystemPrompt, &a.Type, &a.ParentID,
		&toolsJSON, &a.LLMEndpointID, &a.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(toolsJSON), &a.Tools)
	// Parse timestamps leniently; ignore parse errors.
	a.CreatedAt, _ = parseTimestamp(createdAt)
	a.UpdatedAt, _ = parseTimestamp(updatedAt)
	return &a, nil
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// SaveMessage persists a single conversation turn to the messages table.
func (s *Store) SaveMessage(agentID, role, content, toolName, toolInput string, isInterrupted bool) (*MessageRecord, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(
		`INSERT INTO messages (id, agent_id, role, content, tool_name, tool_input, is_interrupted) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, agentID, role, content, nullStr(toolName), nullStr(toolInput), isInterrupted,
	)
	if err != nil {
		return nil, err
	}
	return &MessageRecord{
		ID:            id,
		AgentID:       agentID,
		Role:          role,
		Content:       content,
		ToolName:      toolName,
		ToolInput:     toolInput,
		IsInterrupted: isInterrupted,
		CreatedAt:     time.Now(),
	}, nil
}

// GetMessages returns up to limit messages for the given agent, in
// chronological order.  If before is non-empty it is treated as a cursor
// (message ID) and only messages older than that ID are returned.
func (s *Store) GetMessages(agentID string, limit int, before string) ([]*MessageRecord, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if before == "" {
		rows, err = s.db.Query(
			`SELECT id, agent_id, role, content, COALESCE(tool_name,''), COALESCE(tool_input,''), is_interrupted, created_at
			 FROM messages WHERE agent_id = ? ORDER BY created_at DESC LIMIT ?`,
			agentID, limit,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, agent_id, role, content, COALESCE(tool_name,''), COALESCE(tool_input,''), is_interrupted, created_at
			 FROM messages WHERE agent_id = ?
			   AND created_at < (SELECT created_at FROM messages WHERE id = ?)
			 ORDER BY created_at DESC LIMIT ?`,
			agentID, before, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*MessageRecord
	for rows.Next() {
		var m MessageRecord
		var createdAt string
		if err := rows.Scan(
			&m.ID, &m.AgentID, &m.Role, &m.Content,
			&m.ToolName, &m.ToolInput, &m.IsInterrupted, &createdAt,
		); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = parseTimestamp(createdAt)
		msgs = append(msgs, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Reverse DESC results back to chronological order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// GetSetting returns the stored value for key, or "" if not found.
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting upserts a key/value pair in the settings table.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}

// GetAllSettings returns all settings as a map.
func (s *Store) GetAllSettings() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, rows.Err()
}

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

// CreateEndpoint inserts a new LLM endpoint record.
func (s *Store) CreateEndpoint(name, endpointType, url, apiKey, model string, isDefault bool) (*EndpointRecord, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(
		`INSERT INTO endpoints (id, name, type, url, api_key, model, is_default) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, name, endpointType, url, nullStr(apiKey), model, isDefault,
	)
	if err != nil {
		return nil, err
	}
	return &EndpointRecord{
		ID: id, Name: name, Type: endpointType, URL: url,
		APIKey: apiKey, Model: model, IsDefault: isDefault,
	}, nil
}

// GetDefaultEndpoint returns the endpoint marked as default, if any.
func (s *Store) GetDefaultEndpoint() (*EndpointRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, name, type, url, COALESCE(api_key,''), model, is_default, created_at FROM endpoints WHERE is_default = TRUE LIMIT 1`,
	)
	return scanEndpoint(row)
}

// GetEndpoint fetches a single endpoint by ID.
func (s *Store) GetEndpoint(id string) (*EndpointRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, name, type, url, COALESCE(api_key,''), model, is_default, created_at FROM endpoints WHERE id = ?`,
		id,
	)
	return scanEndpoint(row)
}

// ListEndpoints returns all endpoints in creation order.
func (s *Store) ListEndpoints() ([]*EndpointRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, name, type, url, COALESCE(api_key,''), model, is_default, created_at FROM endpoints ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var eps []*EndpointRecord
	for rows.Next() {
		e, err := scanEndpoint(rows)
		if err != nil {
			return nil, err
		}
		eps = append(eps, e)
	}
	return eps, rows.Err()
}

func scanEndpoint(row scanner) (*EndpointRecord, error) {
	var e EndpointRecord
	var createdAt string
	err := row.Scan(&e.ID, &e.Name, &e.Type, &e.URL, &e.APIKey, &e.Model, &e.IsDefault, &createdAt)
	if err != nil {
		return nil, err
	}
	e.CreatedAt, _ = parseTimestamp(createdAt)
	return &e, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// nullStr converts an empty string to a database NULL value.
func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// parseTimestamp tries several common SQLite timestamp formats.
func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp %q", s)
}
