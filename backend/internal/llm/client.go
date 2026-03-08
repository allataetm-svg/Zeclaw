package llm

import "context"

// ---------------------------------------------------------------------------
// Client interface
// ---------------------------------------------------------------------------

// Client is the interface for all LLM backends.  Both the cloud
// (OpenAI-compatible) and local (Ollama) clients implement it.
type Client interface {
	// ChatCompletion sends a chat request and returns a channel of streaming
	// chunks.  The channel is closed when the response is complete or when an
	// error occurs.  The caller should drain the channel even when an error is
	// detected in a chunk.
	ChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}

// ---------------------------------------------------------------------------
// Shared types
// ---------------------------------------------------------------------------

// Message represents a single entry in the conversation history.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall describes a function invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall holds the name and JSON-encoded arguments of a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDefinition is the OpenAI-format tool descriptor sent to the model.
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition describes one callable function.
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ChatRequest is the input to ChatCompletion.
type ChatRequest struct {
	Model    string
	Messages []Message
	Tools    []map[string]interface{}
	Stream   bool
}

// ChatChunk is a single streaming unit from the model.
type ChatChunk struct {
	Content   string     // partial text delta
	ToolCalls []ToolCall // tool calls from the model (may come in multiple chunks)
	Done      bool       // true on the final sentinel chunk
	Error     error      // non-nil if the stream encountered an error
}

// ---------------------------------------------------------------------------
// Endpoint configuration
// ---------------------------------------------------------------------------

// EndpointConfig holds everything needed to connect to an LLM backend.
type EndpointConfig struct {
	Type   string // "cloud" or "ollama"
	URL    string
	APIKey string
	Model  string
}

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

// NewClient creates the appropriate LLM client based on the endpoint config.
func NewClient(cfg EndpointConfig) Client {
	switch cfg.Type {
	case "ollama":
		return NewOllamaClient(cfg)
	default:
		return NewCloudClient(cfg)
	}
}
