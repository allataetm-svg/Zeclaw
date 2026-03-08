package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/allataetm-svg/zeclaw/backend/internal/llm"
	"github.com/allataetm-svg/zeclaw/backend/internal/protocol"
	"github.com/allataetm-svg/zeclaw/backend/internal/storage"
	"github.com/allataetm-svg/zeclaw/backend/internal/tools"
)

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

// State represents the agent's current lifecycle phase.
type State string

const (
	StateCreated    State = "created"
	StateIdle       State = "idle"
	StateThinking   State = "thinking"
	StateExecuting  State = "executing"
	StateInterrupted State = "interrupted"
	StateError      State = "error"
	StateDeleted    State = "deleted"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// Config holds the immutable settings for an Agent instance.
type Config struct {
	ID           string
	Name         string
	SystemPrompt string
	Type         string // "main" or "sub"
	ParentID     string
	AllowedTools []string
	LLMEndpoint  llm.EndpointConfig
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// Agent runs the LLM think/tool-execute loop for a single conversation.
type Agent struct {
	Config Config
	state  State
	mu     sync.RWMutex

	incomingMessages chan *protocol.Envelope
	interruptChan    chan *protocol.Envelope
	currentCancel    context.CancelFunc

	llmClient    llm.Client
	toolRegistry *tools.Registry
	store        *storage.Store

	// broadcast sends an envelope to all connected WebSocket clients.
	broadcast func(*protocol.Envelope)

	ctx    context.Context
	cancel context.CancelFunc

	// context is the in-memory conversation history sent to the LLM.
	context []llm.Message
}

// New creates a new Agent and wires up its dependencies.
// Call Start() to begin the event loop.
func New(cfg Config, toolRegistry *tools.Registry, store *storage.Store, broadcast func(*protocol.Envelope)) *Agent {
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		Config:           cfg,
		state:            StateCreated,
		incomingMessages: make(chan *protocol.Envelope, 10),
		interruptChan:    make(chan *protocol.Envelope, 1),
		llmClient:        llm.NewClient(cfg.LLMEndpoint),
		toolRegistry:     toolRegistry,
		store:            store,
		broadcast:        broadcast,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start initialises the conversation context and begins the event loop.
func (a *Agent) Start() {
	a.context = []llm.Message{
		{Role: "system", Content: a.Config.SystemPrompt},
	}
	a.setState(StateIdle)
	go a.run()
}

// Stop shuts down the agent's goroutine.
func (a *Agent) Stop() {
	a.cancel()
}

// SendMessage queues an inbound user message envelope for processing.
func (a *Agent) SendMessage(env *protocol.Envelope) {
	select {
	case a.incomingMessages <- env:
	default:
		log.Printf("Agent %s: incoming message channel full, dropping message", a.Config.ID)
	}
}

// Interrupt signals the agent to abort any in-progress work and optionally
// begin a new task from interruptEnv.
func (a *Agent) Interrupt(env *protocol.Envelope) {
	// Replace any queued interrupt with the latest one.
	select {
	case a.interruptChan <- env:
	default:
		select {
		case <-a.interruptChan:
		default:
		}
		a.interruptChan <- env
	}
	// Cancel any in-progress LLM call or tool execution.
	a.mu.Lock()
	if a.currentCancel != nil {
		a.currentCancel()
	}
	a.mu.Unlock()
}

// GetState returns the agent's current state (thread-safe).
func (a *Agent) GetState() State {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// GetLastRunTime is a placeholder; returns current time.
func (a *Agent) GetLastRunTime() time.Time {
	return time.Now()
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

func (a *Agent) setState(s State) {
	a.mu.Lock()
	a.state = s
	a.mu.Unlock()

	// Persist a normalised status string.
	statusStr := "idle"
	switch s {
	case StateThinking, StateExecuting:
		statusStr = "working"
	case StateError:
		statusStr = "error"
	case StateDeleted:
		statusStr = "idle"
	}
	_ = a.store.UpdateAgentStatus(a.Config.ID, statusStr)

	// Build the outbound detail string.
	detail := ""
	switch s {
	case StateThinking:
		detail = "Thinking..."
	case StateExecuting:
		detail = "Executing tool..."
	case StateError:
		detail = "An error occurred"
	case StateInterrupted:
		detail = "Interrupted"
	}

	outStatus := statusStr
	if s == StateInterrupted {
		outStatus = "working"
	}

	env, err := protocol.NewEnvelope(protocol.TypeAgentStatus, a.Config.ID, protocol.AgentStatusPayload{
		Status: outStatus,
		Detail: detail,
	})
	if err == nil {
		a.broadcast(env)
	}
}

// run is the main event loop goroutine.
func (a *Agent) run() {
	log.Printf("Agent %s started (type=%s)", a.Config.ID, a.Config.Type)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Agent %s panicked: %v", a.Config.ID, r)
			a.setState(StateError)
		}
	}()

	for {
		select {
		case <-a.ctx.Done():
			a.setState(StateDeleted)
			return
		case env := <-a.incomingMessages:
			a.processMessage(env)
		}
	}
}

// processMessage handles a TypeUserMessage envelope.
func (a *Agent) processMessage(env *protocol.Envelope) {
	var payload protocol.UserMessagePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		log.Printf("Agent %s: failed to parse user message: %v", a.Config.ID, err)
		return
	}

	// Persist and append to in-memory context.
	_, _ = a.store.SaveMessage(a.Config.ID, "user", payload.Content, "", "", false)
	a.context = append(a.context, llm.Message{Role: "user", Content: payload.Content})

	a.setState(StateThinking)
	a.thinkingLoop()
}

// thinkingLoop runs the agentic loop: LLM → tool calls → LLM → … until
// the model produces a final text response or an interrupt arrives.
func (a *Agent) thinkingLoop() {
	for {
		// Check for a pending interrupt before calling the LLM.
		select {
		case interruptEnv := <-a.interruptChan:
			a.handleInterrupt(interruptEnv)
			return
		case <-a.ctx.Done():
			return
		default:
		}

		// Per-call cancellable context.
		llmCtx, cancel := context.WithCancel(a.ctx)
		a.mu.Lock()
		a.currentCancel = cancel
		a.mu.Unlock()

		toolDefs := a.toolRegistry.ListForAgent(a.Config.AllowedTools)

		chunks, err := a.llmClient.ChatCompletion(llmCtx, llm.ChatRequest{
			Model:    a.Config.LLMEndpoint.Model,
			Messages: a.context,
			Tools:    toolDefs,
			Stream:   true,
		})
		cancel() // release per-call cancel once we have the channel

		if err != nil {
			log.Printf("Agent %s: LLM error: %v", a.Config.ID, err)
			a.setState(StateError)
			a.broadcastError("llm_error", err.Error())
			return
		}

		var fullContent string
		var toolCalls []llm.ToolCall

		// Drain the chunk channel.
		for chunk := range chunks {
			// Check for interrupt during streaming.
			select {
			case interruptEnv := <-a.interruptChan:
				// Drain remaining chunks to avoid goroutine leak.
				go func() {
					for range chunks { //nolint:revive
					}
				}()
				a.handleInterrupt(interruptEnv)
				return
			default:
			}

			if chunk.Error != nil {
				log.Printf("Agent %s: LLM chunk error: %v", a.Config.ID, chunk.Error)
				a.setState(StateError)
				a.broadcastError("llm_error", chunk.Error.Error())
				return
			}

			if chunk.Content != "" {
				fullContent += chunk.Content
				env, _ := protocol.NewEnvelope(protocol.TypeAgentMessage, a.Config.ID, protocol.AgentMessagePayload{
					Content: chunk.Content,
					IsFinal: false,
				})
				a.broadcast(env)
			}
			if len(chunk.ToolCalls) > 0 {
				toolCalls = append(toolCalls, chunk.ToolCalls...)
			}
		}

		if len(toolCalls) > 0 {
			// Model requested tool execution.
			a.setState(StateExecuting)
			a.context = append(a.context, llm.Message{
				Role:      "assistant",
				Content:   fullContent,
				ToolCalls: toolCalls,
			})

			for _, tc := range toolCalls {
				result := a.executeTool(tc)
				a.context = append(a.context, llm.Message{
					Role:       "tool",
					Name:       tc.Function.Name,
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
			a.setState(StateThinking)
			// Loop back to call the LLM with tool results.
			continue
		}

		// No tool calls → model produced its final answer.
		if fullContent != "" {
			_, _ = a.store.SaveMessage(a.Config.ID, "assistant", fullContent, "", "", false)
			a.context = append(a.context, llm.Message{Role: "assistant", Content: fullContent})
		}

		// Send the final sentinel (IsFinal=true, Content="").
		env, _ := protocol.NewEnvelope(protocol.TypeAgentMessage, a.Config.ID, protocol.AgentMessagePayload{
			Content: "",
			IsFinal: true,
		})
		a.broadcast(env)
		a.setState(StateIdle)
		return
	}
}

// executeTool runs a single tool call and returns its string output.
func (a *Agent) executeTool(tc llm.ToolCall) string {
	// Notify clients that the tool is starting.
	startEnv, _ := protocol.NewEnvelope(protocol.TypeToolExecution, a.Config.ID, protocol.ToolExecutionPayload{
		Tool:   tc.Function.Name,
		Input:  tc.Function.Arguments,
		Status: "running",
	})
	a.broadcast(startEnv)

	// Broadcast a detailed status.
	statusEnv, _ := protocol.NewEnvelope(protocol.TypeAgentStatus, a.Config.ID, protocol.AgentStatusPayload{
		Status: "working",
		Detail: fmt.Sprintf("Running %s: %s", tc.Function.Name, tc.Function.Arguments),
	})
	a.broadcast(statusEnv)

	// Parse tool arguments.
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		errMsg := fmt.Sprintf("failed to parse tool arguments: %v", err)
		errEnv, _ := protocol.NewEnvelope(protocol.TypeToolExecution, a.Config.ID, protocol.ToolExecutionPayload{
			Tool:   tc.Function.Name,
			Input:  tc.Function.Arguments,
			Output: errMsg,
			Status: "error",
		})
		a.broadcast(errEnv)
		return errMsg
	}

	// Cancellable context for the tool itself.
	toolCtx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.currentCancel = cancel
	a.mu.Unlock()
	defer cancel()

	output, err := a.toolRegistry.Execute(toolCtx, tc.Function.Name, args)

	status := "done"
	if err != nil {
		status = "error"
		if output == "" {
			output = err.Error()
		}
	}

	// Broadcast tool result.
	doneEnv, _ := protocol.NewEnvelope(protocol.TypeToolExecution, a.Config.ID, protocol.ToolExecutionPayload{
		Tool:   tc.Function.Name,
		Input:  tc.Function.Arguments,
		Output: output,
		Status: status,
	})
	a.broadcast(doneEnv)

	// Persist tool result as a message.
	_, _ = a.store.SaveMessage(a.Config.ID, "tool", output, tc.Function.Name, tc.Function.Arguments, false)

	if output == "" {
		output = "(no output)"
	}
	return output
}

// handleInterrupt aborts the current task and optionally starts a new one.
func (a *Agent) handleInterrupt(interruptEnv *protocol.Envelope) {
	log.Printf("Agent %s: handling interrupt", a.Config.ID)
	a.setState(StateInterrupted)

	var payload protocol.InterruptPayload
	_ = json.Unmarshal(interruptEnv.Payload, &payload)

	// Inject a system notice so the LLM is aware of the interruption.
	a.context = append(a.context, llm.Message{
		Role:    "system",
		Content: "The user interrupted the previous task.",
	})

	// Persist and append the new user message (if any).
	if payload.Content != "" {
		_, _ = a.store.SaveMessage(a.Config.ID, "user", payload.Content, "", "", false)
		a.context = append(a.context, llm.Message{Role: "user", Content: payload.Content})
	}

	// Acknowledge the interrupt.
	ack, _ := protocol.NewEnvelope(protocol.TypeInterruptAck, a.Config.ID, protocol.InterruptAckPayload{
		InterruptedTask: "previous task",
	})
	a.broadcast(ack)

	if payload.Content != "" {
		a.setState(StateThinking)
		a.thinkingLoop()
	} else {
		a.setState(StateIdle)
	}
}

// broadcastError sends a TypeError envelope to all clients.
func (a *Agent) broadcastError(code, message string) {
	env, _ := protocol.NewEnvelope(protocol.TypeError, a.Config.ID, protocol.ErrorPayload{
		Code:    code,
		Message: message,
	})
	a.broadcast(env)
}
