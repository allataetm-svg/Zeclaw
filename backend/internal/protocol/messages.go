package protocol

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Envelope
// ---------------------------------------------------------------------------

// Envelope is the top-level wrapper for every WebSocket message (both
// inbound from the Android client and outbound from the backend).
type Envelope struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	AgentID   string          `json:"agent_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// Marshal serialises the envelope to JSON.
func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// NewEnvelope creates a populated Envelope, marshalling payload into
// json.RawMessage. A new UUID and the current time are set automatically.
func NewEnvelope(msgType string, agentID string, payload interface{}) (*Envelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		ID:        uuid.New().String(),
		Type:      msgType,
		AgentID:   agentID,
		Timestamp: time.Now(),
		Payload:   json.RawMessage(raw),
	}, nil
}

// ---------------------------------------------------------------------------
// Inbound message type constants
// ---------------------------------------------------------------------------

const (
	TypeUserMessage     = "user_message"
	TypeInterrupt       = "interrupt"
	TypeCreateAgent     = "create_agent"
	TypeDeleteAgent     = "delete_agent"
	TypeStopAgent       = "stop_agent"
	TypeRetryAgent      = "retry_agent"
	TypeUpdateSettings  = "update_settings"
	TypeGetAgents       = "get_agents"
	TypeGetHistory      = "get_history"
	TypeCreateCron      = "create_cron"
	TypeDeleteCron      = "delete_cron"
	TypeUpdateCron      = "update_cron"
	TypeGetCrons        = "get_crons"
	TypeUpdateHeartbeat = "update_heartbeat"
	TypeMemorySearch    = "memory_search"
	TypeRunDoctor       = "run_doctor"
	TypeApproveAction   = "approve_action"
	TypeGetAuditLog     = "get_audit_log"
	TypeUpdateSkill     = "update_skill"
	TypeGetSkills       = "get_skills"
)

// ---------------------------------------------------------------------------
// Outbound message type constants
// ---------------------------------------------------------------------------

const (
	TypeAgentMessage    = "agent_message"
	TypeAgentStatus     = "agent_status"
	TypeToolExecution   = "tool_execution"
	TypeAgentList       = "agent_list"
	TypeHistory         = "history"
	TypeError           = "error"
	TypeInterruptAck    = "interrupt_ack"
	TypeSettingsUpdated = "settings_updated"
	TypeHeartbeatResult = "heartbeat_result"
	TypeCronTriggered   = "cron_triggered"
	TypeCronList        = "cron_list"
	TypeMemoryResults   = "memory_results"
	TypeDoctorReport    = "doctor_report"
	TypeApprovalRequest = "approval_request"
	TypeAuditLog        = "audit_log"
	TypeSkillList       = "skill_list"
	TypeHookFired       = "hook_fired"
)

// ---------------------------------------------------------------------------
// Payload structs
// ---------------------------------------------------------------------------

// UserMessagePayload is the inbound payload for TypeUserMessage.
type UserMessagePayload struct {
	Content string `json:"content"`
}

// AgentMessagePayload is the outbound payload for TypeAgentMessage.
// IsFinal is true on the last chunk (or a no-content final signal).
type AgentMessagePayload struct {
	Content string `json:"content"`
	IsFinal bool   `json:"is_final"`
}

// AgentStatusPayload is the outbound payload for TypeAgentStatus.
// Status is one of "idle", "working", "error".
type AgentStatusPayload struct {
	Status string `json:"status"` // "idle" | "working" | "error"
	Detail string `json:"detail"`
}

// ToolExecutionPayload is the outbound payload for TypeToolExecution.
// Status is one of "running", "done", "error".
type ToolExecutionPayload struct {
	Tool   string `json:"tool"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Status string `json:"status"` // "running" | "done" | "error"
}

// CreateAgentPayload is the inbound payload for TypeCreateAgent.
type CreateAgentPayload struct {
	Name          string   `json:"name"`
	SystemPrompt  string   `json:"system_prompt"`
	Type          string   `json:"type"` // "main" | "sub"
	ParentID      string   `json:"parent_id,omitempty"`
	Tools         []string `json:"tools"`
	LLMEndpointID string   `json:"llm_endpoint_id,omitempty"`
}

// AgentListPayload is the outbound payload for TypeAgentList.
type AgentListPayload struct {
	Agents []AgentInfo `json:"agents"`
}

// AgentInfo is a compact agent descriptor used inside AgentListPayload.
type AgentInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ParentID string `json:"parent_id,omitempty"`
	Status   string `json:"status"`
}

// ErrorPayload is the outbound payload for TypeError.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GetHistoryPayload is the inbound payload for TypeGetHistory.
type GetHistoryPayload struct {
	Limit  int    `json:"limit"`
	Before string `json:"before,omitempty"` // cursor (message ID)
}

// HistoryPayload is the outbound payload for TypeHistory.
type HistoryPayload struct {
	Messages []StoredMessage `json:"messages"`
	Cursor   string          `json:"cursor,omitempty"` // ID of the oldest returned message
}

// StoredMessage is a single persisted message returned in HistoryPayload.
type StoredMessage struct {
	ID            string    `json:"id"`
	AgentID       string    `json:"agent_id"`
	Role          string    `json:"role"`
	Content       string    `json:"content"`
	ToolName      string    `json:"tool_name,omitempty"`
	ToolInput     string    `json:"tool_input,omitempty"`
	IsInterrupted bool      `json:"is_interrupted"`
	CreatedAt     time.Time `json:"created_at"`
}

// UpdateSettingsPayload is the inbound payload for TypeUpdateSettings.
type UpdateSettingsPayload struct {
	Settings map[string]interface{} `json:"settings"`
}

// SettingsUpdatedPayload is the outbound payload for TypeSettingsUpdated.
type SettingsUpdatedPayload struct {
	Settings map[string]interface{} `json:"settings"`
}

// InterruptAckPayload is the outbound payload for TypeInterruptAck.
type InterruptAckPayload struct {
	InterruptedTask string `json:"interrupted_task"`
}

// InterruptPayload is the inbound payload for TypeInterrupt.
type InterruptPayload struct {
	Content string `json:"content,omitempty"`
}
