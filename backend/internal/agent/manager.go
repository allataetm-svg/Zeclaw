package agent

import (
	"fmt"
	"sync"

	"github.com/allataetm-svg/zeclaw/backend/internal/llm"
	"github.com/allataetm-svg/zeclaw/backend/internal/protocol"
	"github.com/allataetm-svg/zeclaw/backend/internal/storage"
	"github.com/allataetm-svg/zeclaw/backend/internal/tools"
)

// ---------------------------------------------------------------------------
// Manager
// ---------------------------------------------------------------------------

// Manager owns all running Agent instances and provides a thread-safe API
// for creating, routing messages to, and deleting agents.
type Manager struct {
	mu           sync.RWMutex
	agents       map[string]*Agent
	toolRegistry *tools.Registry
	store        *storage.Store
	broadcast    func(*protocol.Envelope)
}

// NewManager creates a Manager backed by the given store and tool registry.
// The broadcast function is forwarded to every Agent so it can push updates
// to all WebSocket clients.
func NewManager(toolRegistry *tools.Registry, store *storage.Store, broadcast func(*protocol.Envelope)) *Manager {
	return &Manager{
		agents:       make(map[string]*Agent),
		toolRegistry: toolRegistry,
		store:        store,
		broadcast:    broadcast,
	}
}

// LoadAgentsFromDB reads all persisted agents from the database, resolves
// their LLM endpoints, and starts each agent's event loop.
func (m *Manager) LoadAgentsFromDB() error {
	agents, err := m.store.ListAgents()
	if err != nil {
		return err
	}
	for _, a := range agents {
		endpointCfg := m.resolveEndpoint(a.LLMEndpointID)
		cfg := Config{
			ID:           a.ID,
			Name:         a.Name,
			SystemPrompt: a.SystemPrompt,
			Type:         a.Type,
			ParentID:     a.ParentID,
			AllowedTools: a.Tools,
			LLMEndpoint:  endpointCfg,
		}
		ag := New(cfg, m.toolRegistry, m.store, m.broadcast)
		ag.Start()

		m.mu.Lock()
		m.agents[a.ID] = ag
		m.mu.Unlock()
	}
	return nil
}

// CreateAgent persists a new agent, starts it, and returns the Agent instance.
// Only one "main" type agent is allowed at a time.
func (m *Manager) CreateAgent(name, systemPrompt, agentType, parentID, endpointID string, allowedTools []string) (*Agent, error) {
	if agentType == "main" {
		m.mu.RLock()
		for _, a := range m.agents {
			if a.Config.Type == "main" {
				m.mu.RUnlock()
				return nil, fmt.Errorf("a main agent already exists")
			}
		}
		m.mu.RUnlock()
	}

	record, err := m.store.CreateAgent(name, systemPrompt, agentType, parentID, endpointID, allowedTools)
	if err != nil {
		return nil, fmt.Errorf("saving agent: %w", err)
	}

	endpointCfg := m.resolveEndpoint(endpointID)
	cfg := Config{
		ID:           record.ID,
		Name:         record.Name,
		SystemPrompt: record.SystemPrompt,
		Type:         record.Type,
		ParentID:     record.ParentID,
		AllowedTools: record.Tools,
		LLMEndpoint:  endpointCfg,
	}

	ag := New(cfg, m.toolRegistry, m.store, m.broadcast)
	ag.Start()

	m.mu.Lock()
	m.agents[record.ID] = ag
	m.mu.Unlock()

	return ag, nil
}

// GetAgent looks up a running agent by ID.
func (m *Manager) GetAgent(id string) (*Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.agents[id]
	return a, ok
}

// ListAgents returns a snapshot of all running agents.
func (m *Manager) ListAgents() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Agent, 0, len(m.agents))
	for _, a := range m.agents {
		list = append(list, a)
	}
	return list
}

// DeleteAgent stops the agent, removes it from the in-memory map, and
// deletes its database record.
func (m *Manager) DeleteAgent(id string) error {
	m.mu.Lock()
	ag, ok := m.agents[id]
	if ok {
		ag.Stop()
		delete(m.agents, id)
	}
	m.mu.Unlock()

	return m.store.DeleteAgent(id)
}

// RouteMessage delivers an inbound message envelope to the named agent.
func (m *Manager) RouteMessage(agentID string, env *protocol.Envelope) error {
	ag, ok := m.GetAgent(agentID)
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	ag.SendMessage(env)
	return nil
}

// InterruptAgent sends an interrupt signal to the named agent.
func (m *Manager) InterruptAgent(agentID string, env *protocol.Envelope) error {
	ag, ok := m.GetAgent(agentID)
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}
	ag.Interrupt(env)
	return nil
}

// BroadcastAgentList pushes a fresh TypeAgentList envelope to all clients.
func (m *Manager) BroadcastAgentList() {
	agents := m.ListAgents()
	infos := make([]protocol.AgentInfo, 0, len(agents))
	for _, a := range agents {
		infos = append(infos, protocol.AgentInfo{
			ID:       a.Config.ID,
			Name:     a.Config.Name,
			Type:     a.Config.Type,
			ParentID: a.Config.ParentID,
			Status:   string(a.GetState()),
		})
	}
	env, _ := protocol.NewEnvelope(protocol.TypeAgentList, "", protocol.AgentListPayload{
		Agents: infos,
	})
	m.broadcast(env)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolveEndpoint returns an EndpointConfig for the given ID, falling back
// to the default endpoint if the ID is empty or not found.
func (m *Manager) resolveEndpoint(endpointID string) llm.EndpointConfig {
	var ep *storage.EndpointRecord
	if endpointID != "" {
		ep, _ = m.store.GetEndpoint(endpointID)
	}
	if ep == nil {
		ep, _ = m.store.GetDefaultEndpoint()
	}
	if ep == nil {
		return llm.EndpointConfig{}
	}
	return llm.EndpointConfig{
		Type:   ep.Type,
		URL:    ep.URL,
		APIKey: ep.APIKey,
		Model:  ep.Model,
	}
}
