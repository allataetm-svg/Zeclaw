package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ---------------------------------------------------------------------------
// CloudClient – OpenAI-compatible SSE streaming client
// ---------------------------------------------------------------------------

// CloudClient talks to any OpenAI-compatible /v1/chat/completions endpoint.
type CloudClient struct {
	cfg        EndpointConfig
	httpClient *http.Client
}

// NewCloudClient constructs a CloudClient for the given endpoint.
func NewCloudClient(cfg EndpointConfig) *CloudClient {
	return &CloudClient{
		cfg:        cfg,
		httpClient: &http.Client{},
	}
}

// ---------------------------------------------------------------------------
// Wire types (OpenAI format)
// ---------------------------------------------------------------------------

type cloudRequest struct {
	Model    string                   `json:"model"`
	Messages []cloudMessage           `json:"messages"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
	Stream   bool                     `json:"stream"`
}

type cloudMessage struct {
	Role       string          `json:"role"`
	Content    interface{}     `json:"content"` // string or null
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []cloudToolCall `json:"tool_calls,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type cloudToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function cloudFunctionCall  `json:"function"`
}

type cloudFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// SSE delta types

type sseResponse struct {
	Choices []sseChoice `json:"choices"`
}

type sseChoice struct {
	Delta        sseDelta `json:"delta"`
	FinishReason string   `json:"finish_reason"`
}

type sseDelta struct {
	Role      string          `json:"role"`
	Content   *string         `json:"content"`
	ToolCalls []sseToolCall   `json:"tool_calls"`
}

type sseToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function sseFunctionCall    `json:"function"`
}

type sseFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ---------------------------------------------------------------------------
// ChatCompletion
// ---------------------------------------------------------------------------

// ChatCompletion sends a streaming chat request and returns a channel of
// chunks.  The channel is closed when the stream ends or the context is
// cancelled.
func (c *CloudClient) ChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	// Build request body
	msgs := make([]cloudMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		cm := cloudMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				cm.ToolCalls = append(cm.ToolCalls, cloudToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: cloudFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			// When there are tool_calls the content should be null/empty.
			cm.Content = nil
		}
		msgs = append(msgs, cm)
	}

	body := cloudRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   true,
	}
	if len(req.Tools) > 0 {
		body.Tools = req.Tools
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("cloud: marshalling request: %w", err)
	}

	url := strings.TrimRight(c.cfg.URL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("cloud: building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cloud: HTTP request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("cloud: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan ChatChunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		c.parseSSE(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// parseSSE reads the SSE stream from r and sends chunks to ch.
func (c *CloudClient) parseSSE(ctx context.Context, r io.Reader, ch chan<- ChatChunk) {
	// toolCallAccum accumulates fragments from multiple SSE deltas that
	// together form a single tool call.
	type accumEntry struct {
		id        string
		toolType  string
		name      string
		arguments strings.Builder
	}
	toolAccum := make(map[int]*accumEntry)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// Honour context cancellation.
		select {
		case <-ctx.Done():
			ch <- ChatChunk{Error: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()

		// SSE events are prefixed with "data: ".
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			ch <- ChatChunk{Done: true}
			return
		}

		var evt sseResponse
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			// Malformed SSE line – skip silently.
			continue
		}

		if len(evt.Choices) == 0 {
			continue
		}

		delta := evt.Choices[0].Delta

		// Text content.
		if delta.Content != nil && *delta.Content != "" {
			ch <- ChatChunk{Content: *delta.Content}
		}

		// Tool call fragments – accumulate by index.
		for _, tc := range delta.ToolCalls {
			entry, exists := toolAccum[tc.Index]
			if !exists {
				entry = &accumEntry{}
				toolAccum[tc.Index] = entry
			}
			if tc.ID != "" {
				entry.id = tc.ID
			}
			if tc.Type != "" {
				entry.toolType = tc.Type
			}
			if tc.Function.Name != "" {
				entry.name = tc.Function.Name
			}
			entry.arguments.WriteString(tc.Function.Arguments)
		}

		// On stop, flush accumulated tool calls.
		if evt.Choices[0].FinishReason == "tool_calls" || evt.Choices[0].FinishReason == "stop" {
			if len(toolAccum) > 0 {
				tcs := make([]ToolCall, 0, len(toolAccum))
				for _, e := range toolAccum {
					tcs = append(tcs, ToolCall{
						ID:   e.id,
						Type: e.toolType,
						Function: FunctionCall{
							Name:      e.name,
							Arguments: e.arguments.String(),
						},
					})
				}
				ch <- ChatChunk{ToolCalls: tcs}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- ChatChunk{Error: fmt.Errorf("cloud: reading SSE stream: %w", err)}
		return
	}

	// Stream ended without explicit [DONE] – flush any accumulated tool calls.
	if len(toolAccum) > 0 {
		tcs := make([]ToolCall, 0, len(toolAccum))
		for _, e := range toolAccum {
			tcs = append(tcs, ToolCall{
				ID:   e.id,
				Type: e.toolType,
				Function: FunctionCall{
					Name:      e.name,
					Arguments: e.arguments.String(),
				},
			})
		}
		ch <- ChatChunk{ToolCalls: tcs}
	}
	ch <- ChatChunk{Done: true}
}
