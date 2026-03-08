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
// OllamaClient – NDJSON streaming client for Ollama /api/chat
// ---------------------------------------------------------------------------

// OllamaClient talks to a locally-running Ollama instance.
type OllamaClient struct {
	cfg        EndpointConfig
	httpClient *http.Client
}

// NewOllamaClient constructs an OllamaClient for the given endpoint.
func NewOllamaClient(cfg EndpointConfig) *OllamaClient {
	return &OllamaClient{
		cfg:        cfg,
		httpClient: &http.Client{},
	}
}

// ---------------------------------------------------------------------------
// Wire types (Ollama format)
// ---------------------------------------------------------------------------

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
}

type ollamaMessage struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	ToolCalls []ollamaToolCall  `json:"tool_calls,omitempty"`
}

type ollamaTool struct {
	Type     string              `json:"type"`
	Function ollamaFunctionDef   `json:"function"`
}

type ollamaFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ollamaToolCall struct {
	Function ollamaFunctionCall `json:"function"`
}

type ollamaFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// NDJSON response types

type ollamaResponse struct {
	Model   string         `json:"model"`
	Message *ollamaMessage `json:"message"`
	Done    bool           `json:"done"`
	Error   string         `json:"error"`
}

// ---------------------------------------------------------------------------
// ChatCompletion
// ---------------------------------------------------------------------------

// ChatCompletion sends a streaming chat request to Ollama and returns a
// channel of chunks.
func (c *OllamaClient) ChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	// Convert messages to Ollama format.
	msgs := make([]ollamaMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		om := ollamaMessage{
			Role:    m.Role,
			Content: m.Content,
		}
		// Represent tool calls as structured objects.
		for _, tc := range m.ToolCalls {
			// Ollama expects arguments as a map; try to unmarshal the JSON string.
			var argsMap map[string]interface{}
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &argsMap)
			om.ToolCalls = append(om.ToolCalls, ollamaToolCall{
				Function: ollamaFunctionCall{
					Name:      tc.Function.Name,
					Arguments: argsMap,
				},
			})
		}
		msgs = append(msgs, om)
	}

	// Convert tools to Ollama format.
	var ollamaTools []ollamaTool
	for _, t := range req.Tools {
		fn, ok := t["function"].(map[string]interface{})
		if !ok {
			continue
		}
		params, _ := fn["parameters"].(map[string]interface{})
		ollamaTools = append(ollamaTools, ollamaTool{
			Type: "function",
			Function: ollamaFunctionDef{
				Name:        fmt.Sprintf("%v", fn["name"]),
				Description: fmt.Sprintf("%v", fn["description"]),
				Parameters:  params,
			},
		})
	}

	body := ollamaRequest{
		Model:    req.Model,
		Messages: msgs,
		Stream:   true,
		Tools:    ollamaTools,
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshalling request: %w", err)
	}

	url := strings.TrimRight(c.cfg.URL, "/") + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("ollama: building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: HTTP request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan ChatChunk, 32)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		c.parseNDJSON(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// parseNDJSON reads the NDJSON stream and sends chunks to ch.
func (c *OllamaClient) parseNDJSON(ctx context.Context, r io.Reader, ch chan<- ChatChunk) {
	scanner := bufio.NewScanner(r)
	// Increase the scanner buffer to handle large JSON lines.
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- ChatChunk{Error: ctx.Err()}
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var evt ollamaResponse
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			// Skip malformed lines.
			continue
		}

		if evt.Error != "" {
			ch <- ChatChunk{Error: fmt.Errorf("ollama: %s", evt.Error)}
			return
		}

		if evt.Message != nil {
			// Text content delta.
			if evt.Message.Content != "" {
				ch <- ChatChunk{Content: evt.Message.Content}
			}

			// Tool calls – convert back to canonical ToolCall format.
			if len(evt.Message.ToolCalls) > 0 {
				tcs := make([]ToolCall, 0, len(evt.Message.ToolCalls))
				for _, otc := range evt.Message.ToolCalls {
					argsJSON, _ := json.Marshal(otc.Function.Arguments)
					tcs = append(tcs, ToolCall{
						ID:   "", // Ollama may not provide an ID
						Type: "function",
						Function: FunctionCall{
							Name:      otc.Function.Name,
							Arguments: string(argsJSON),
						},
					})
				}
				ch <- ChatChunk{ToolCalls: tcs}
			}
		}

		if evt.Done {
			ch <- ChatChunk{Done: true}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- ChatChunk{Error: fmt.Errorf("ollama: reading NDJSON stream: %w", err)}
		return
	}

	// Stream ended without an explicit done=true; send final sentinel.
	ch <- ChatChunk{Done: true}
}
