package server

import (
	"encoding/json"
	"log"

	"github.com/allataetm-svg/zeclaw/backend/internal/protocol"
)

// handleMessage deserialises the raw WebSocket frame and dispatches it to
// the appropriate handler based on the envelope type.
func (s *Server) handleMessage(client *Client, data []byte) {
	var env protocol.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		log.Printf("Failed to parse message: %v", err)
		s.sendError(client, "parse_error", "invalid message format")
		return
	}

	switch env.Type {
	case protocol.TypeGetAgents:
		s.handleGetAgents(client, &env)
	case protocol.TypeCreateAgent:
		s.handleCreateAgent(client, &env)
	case protocol.TypeDeleteAgent:
		s.handleDeleteAgent(client, &env)
	case protocol.TypeUserMessage:
		s.handleUserMessage(client, &env)
	case protocol.TypeInterrupt:
		s.handleInterrupt(client, &env)
	case protocol.TypeStopAgent:
		s.handleStopAgent(client, &env)
	case protocol.TypeGetHistory:
		s.handleGetHistory(client, &env)
	case protocol.TypeUpdateSettings:
		s.handleUpdateSettings(client, &env)
	default:
		log.Printf("Unhandled message type: %s", env.Type)
	}
}

// ---------------------------------------------------------------------------
// Agent management
// ---------------------------------------------------------------------------

func (s *Server) handleGetAgents(client *Client, env *protocol.Envelope) {
	agents := s.agentMgr.ListAgents()
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
	resp, err := protocol.NewEnvelope(protocol.TypeAgentList, "", protocol.AgentListPayload{
		Agents: infos,
	})
	if err != nil {
		return
	}
	s.SendToClient(client, resp)
}

func (s *Server) handleCreateAgent(client *Client, env *protocol.Envelope) {
	var payload protocol.CreateAgentPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		s.sendError(client, "parse_error", "invalid create_agent payload")
		return
	}

	agentType := payload.Type
	if agentType == "" {
		agentType = "sub"
	}

	_, err := s.agentMgr.CreateAgent(
		payload.Name,
		payload.SystemPrompt,
		agentType,
		payload.ParentID,
		payload.LLMEndpointID,
		payload.Tools,
	)
	if err != nil {
		s.sendError(client, "create_agent_error", err.Error())
		return
	}

	// Push an updated agent list to all connected clients.
	s.agentMgr.BroadcastAgentList()
}

func (s *Server) handleDeleteAgent(client *Client, env *protocol.Envelope) {
	if env.AgentID == "" {
		s.sendError(client, "missing_agent_id", "agent_id is required")
		return
	}
	if err := s.agentMgr.DeleteAgent(env.AgentID); err != nil {
		s.sendError(client, "delete_agent_error", err.Error())
		return
	}
	s.agentMgr.BroadcastAgentList()
}

// ---------------------------------------------------------------------------
// Messaging
// ---------------------------------------------------------------------------

func (s *Server) handleUserMessage(client *Client, env *protocol.Envelope) {
	if env.AgentID == "" {
		s.sendError(client, "missing_agent_id", "agent_id is required")
		return
	}
	if err := s.agentMgr.RouteMessage(env.AgentID, env); err != nil {
		s.sendError(client, "route_error", err.Error())
	}
}

func (s *Server) handleInterrupt(client *Client, env *protocol.Envelope) {
	if env.AgentID == "" {
		s.sendError(client, "missing_agent_id", "agent_id is required")
		return
	}
	if err := s.agentMgr.InterruptAgent(env.AgentID, env); err != nil {
		s.sendError(client, "interrupt_error", err.Error())
	}
}

func (s *Server) handleStopAgent(client *Client, env *protocol.Envelope) {
	if env.AgentID == "" {
		s.sendError(client, "missing_agent_id", "agent_id is required")
		return
	}
	// A "stop" is an interrupt with no follow-up content.
	emptyInterrupt, _ := protocol.NewEnvelope(protocol.TypeInterrupt, env.AgentID, protocol.InterruptPayload{
		Content: "",
	})
	if err := s.agentMgr.InterruptAgent(env.AgentID, emptyInterrupt); err != nil {
		s.sendError(client, "stop_agent_error", err.Error())
	}
}

// ---------------------------------------------------------------------------
// History
// ---------------------------------------------------------------------------

func (s *Server) handleGetHistory(client *Client, env *protocol.Envelope) {
	if env.AgentID == "" {
		s.sendError(client, "missing_agent_id", "agent_id is required")
		return
	}

	var payload protocol.GetHistoryPayload
	// Errors here are non-fatal; we fall back to defaults.
	_ = json.Unmarshal(env.Payload, &payload)
	if payload.Limit <= 0 {
		payload.Limit = 50
	}

	msgs, err := s.store.GetMessages(env.AgentID, payload.Limit, payload.Before)
	if err != nil {
		s.sendError(client, "history_error", err.Error())
		return
	}

	storedMsgs := make([]protocol.StoredMessage, 0, len(msgs))
	for _, m := range msgs {
		storedMsgs = append(storedMsgs, protocol.StoredMessage{
			ID:            m.ID,
			AgentID:       m.AgentID,
			Role:          m.Role,
			Content:       m.Content,
			ToolName:      m.ToolName,
			ToolInput:     m.ToolInput,
			IsInterrupted: m.IsInterrupted,
			CreatedAt:     m.CreatedAt,
		})
	}

	// Cursor points to the oldest returned message so the client can page
	// backwards.
	cursor := ""
	if len(storedMsgs) > 0 {
		cursor = storedMsgs[0].ID
	}

	resp, _ := protocol.NewEnvelope(protocol.TypeHistory, env.AgentID, protocol.HistoryPayload{
		Messages: storedMsgs,
		Cursor:   cursor,
	})
	s.SendToClient(client, resp)
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

func (s *Server) handleUpdateSettings(client *Client, env *protocol.Envelope) {
	var payload protocol.UpdateSettingsPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		s.sendError(client, "parse_error", "invalid update_settings payload")
		return
	}

	for k, v := range payload.Settings {
		val, _ := json.Marshal(v)
		if err := s.store.SetSetting(k, string(val)); err != nil {
			s.sendError(client, "settings_error", err.Error())
			return
		}
	}

	resp, _ := protocol.NewEnvelope(protocol.TypeSettingsUpdated, "", protocol.SettingsUpdatedPayload{
		Settings: payload.Settings,
	})
	s.SendToClient(client, resp)
}

// ---------------------------------------------------------------------------
// Error helper
// ---------------------------------------------------------------------------

func (s *Server) sendError(client *Client, code, message string) {
	env, _ := protocol.NewEnvelope(protocol.TypeError, "", protocol.ErrorPayload{
		Code:    code,
		Message: message,
	})
	s.SendToClient(client, env)
}
