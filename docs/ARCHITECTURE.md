# Zeclaw - System Architecture Document

---

## Table of Contents

1. [High-Level Architecture](#1-high-level-architecture)
2. [Component Overview](#2-component-overview)
3. [Communication Layer](#3-communication-layer)
4. [Agent State Machine](#4-agent-state-machine)
5. [Interruptible Execution Model](#5-interruptible-execution-model)
6. [Multi-Agent Management](#6-multi-agent-management)
7. [Tool Execution Framework](#7-tool-execution-framework)
8. [LLM Integration Layer](#8-llm-integration-layer)
9. [Data Flow Diagrams](#9-data-flow-diagrams)
10. [Security Model](#10-security-model)
11. [Data Persistence](#11-data-persistence)
12. [Error Handling Strategy](#12-error-handling-strategy)

---

## 1. High-Level Architecture

Zeclaw runs entirely on-device. The Flutter frontend communicates with a local Go binary backend over a WebSocket connection bound to `localhost`.

```
┌──────────────────────────────────────────────────────┐
│                    Android Device                     │
│                                                      │
│  ┌─────────────────────┐   WebSocket    ┌──────────────────────┐
│  │                     │  (localhost)    │                      │
│  │   Flutter Frontend  │◄──────────────►│   Go Backend Binary  │
│  │                     │   :8085        │                      │
│  │  ┌───────────────┐  │                │  ┌────────────────┐  │
│  │  │ Chat UI       │  │                │  │ Agent Engine   │  │
│  │  │ Dashboard UI  │  │                │  │  ├─ State Mgr  │  │
│  │  │ Settings UI   │  │                │  │  ├─ Tool Exec  │  │
│  │  │ WS Client     │  │                │  │  └─ Interrupt  │  │
│  │  └───────────────┘  │                │  ├────────────────┤  │
│  │                     │                │  │ LLM Connector  │  │
│  │                     │                │  │  ├─ Cloud API  │  │
│  │                     │                │  │  └─ Ollama     │  │
│  │                     │                │  ├────────────────┤  │
│  │                     │                │  │ Persistence    │  │
│  │                     │                │  │  └─ SQLite     │  │
│  └─────────────────────┘                │  └────────────────┘  │
│                                          │                      │
│                                          │  ┌────────────────┐  │
│                                          │  │ Tools          │  │
│                                          │  │  ├─ Shell      │  │
│                                          │  │  └─ ReadFile   │  │
│                                          │  └────────────────┘  │
│                                          │                      │
│                                          └──────────────────────┘
│                                                      │
│                              ┌────────────────────────┘
│                              │ (optional)
│                              ▼
│                    ┌──────────────────┐
│                    │  Local Ollama    │
│                    │  (localhost)     │
│                    └──────────────────┘
│                                                      │
└──────────────────────────────────────────────────────┘
              │
              │ HTTPS (outbound only)
              ▼
    ┌──────────────────┐
    │  Cloud LLM APIs  │
    │  (OpenAI, etc.)  │
    └──────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Local Go binary** (not remote server) | All computation stays on-device. No cloud dependency for core logic. Low latency. Privacy-first. |
| **WebSocket** (not REST) | Required for real-time bidirectional streaming. Agent status updates and user interrupts must flow instantly. |
| **SQLite** (not server DB) | Lightweight, embedded, zero-config. Perfect for on-device persistence. |
| **LLM-agnostic** | Users choose their provider. No vendor lock-in. Support both cloud and local inference. |

---

## 2. Component Overview

### 2.1 Flutter Frontend

| Component           | Responsibility                                                                  |
|---------------------|---------------------------------------------------------------------------------|
| **WebSocket Client**| Maintains persistent connection to Go backend. Handles reconnection, message serialization/deserialization. |
| **State Management**| Manages UI state using Riverpod. Stores agent list, active agent, messages, dashboard state. |
| **Chat Controller** | Manages message list, streaming text rendering, interrupt signaling.            |
| **Dashboard Controller** | Polls/receives agent status updates, renders task progress.                |
| **Settings Controller** | CRUD operations for LLM endpoints, app preferences. Persisted via backend. |
| **Router**          | Bottom tab navigation using GoRouter. Deep linking to specific agent chats.     |

### 2.2 Go Backend

| Component           | Responsibility                                                                  |
|---------------------|---------------------------------------------------------------------------------|
| **WebSocket Server**| Listens on `localhost:8085`. Handles multiple concurrent connections. Routes messages to appropriate handlers. |
| **Agent Manager**   | Creates, stores, and orchestrates agents. Maintains the registry of all agents. |
| **Agent Runtime**   | Per-agent goroutine managing the agent's execution loop, context, and state transitions. |
| **Interrupt Handler**| Listens for interrupt signals on a per-agent channel. Triggers cancellation of the current execution context. |
| **Tool Registry**   | Registers available tools. Dispatches tool calls from the LLM to the correct executor. |
| **Tool Executors**  | `ShellExecutor`: runs shell commands via `os/exec`. `ReadFileExecutor`: reads file contents. |
| **LLM Client**      | Abstracts LLM communication. Supports OpenAI-compatible REST APIs and Ollama API. Handles streaming responses. |
| **Persistence Layer**| SQLite database for conversations, agent configs, settings. Uses `modernc.org/sqlite` (pure Go, no CGO). |
| **Status Broadcaster**| Publishes granular status updates to the WebSocket for real-time frontend rendering. |

---

## 3. Communication Layer

### 3.1 WebSocket Protocol

All communication between Flutter and Go uses a single WebSocket connection over `ws://localhost:8085/ws`.

#### Message Envelope

Every message follows a unified envelope format:

```json
{
  "id": "msg_uuid_v4",
  "type": "message_type",
  "agent_id": "agent_uuid",
  "timestamp": "2026-03-08T12:00:00Z",
  "payload": { }
}
```

#### Message Types (Frontend → Backend)

| Type                  | Payload                                              | Description                              |
|-----------------------|------------------------------------------------------|------------------------------------------|
| `user_message`        | `{ "content": "..." }`                               | User sends a chat message to an agent.   |
| `interrupt`           | `{ "content": "..." }`                               | User interrupts agent with a new message.|
| `create_agent`        | `{ "name": "...", "system_prompt": "...", "type": "main|sub", "parent_id": "...", "tools": [...], "llm_endpoint_id": "..." }` | Create a new agent. |
| `delete_agent`        | `{ }`                                                | Delete an agent.                         |
| `stop_agent`          | `{ }`                                                | Force-stop an agent's current task.      |
| `retry_agent`         | `{ }`                                                | Retry last failed task.                  |
| `update_settings`     | `{ "settings": { ... } }`                            | Update app/LLM settings.                 |
| `get_agents`          | `{ }`                                                | Request list of all agents.              |
| `get_history`         | `{ "limit": 50, "before": "cursor" }`               | Fetch conversation history.              |

#### Message Types (Backend → Frontend)

| Type                  | Payload                                              | Description                              |
|-----------------------|------------------------------------------------------|------------------------------------------|
| `agent_message`       | `{ "content": "...", "is_final": bool }`             | Agent's streamed response text.          |
| `agent_status`        | `{ "status": "idle|working|error", "detail": "..." }`| Real-time status update.                |
| `tool_execution`      | `{ "tool": "shell|read_file", "input": "...", "output": "...", "status": "running|done|error" }` | Tool execution progress. |
| `agent_list`          | `{ "agents": [...] }`                                | Response to `get_agents`.                |
| `history`             | `{ "messages": [...], "cursor": "..." }`             | Response to `get_history`.               |
| `error`               | `{ "code": "...", "message": "..." }`                | Error response.                          |
| `interrupt_ack`       | `{ "interrupted_task": "..." }`                      | Acknowledgment that interrupt was received.|
| `settings_updated`    | `{ "settings": { ... } }`                            | Confirmation of settings update.         |

### 3.2 Connection Lifecycle

```
Flutter App Start
  │
  ├─ Start Go binary as a subprocess (via platform channel)
  │    └─ Go binary starts WebSocket server on localhost:8085
  │
  ├─ Flutter WS client connects to ws://localhost:8085/ws
  │    ├─ On connect: send `get_agents` to load state
  │    └─ On connect: send `get_history` for active agent
  │
  ├─ Connection maintained with ping/pong heartbeat (30s interval)
  │
  ├─ On disconnect:
  │    ├─ Flutter shows "Reconnecting..." banner
  │    ├─ Exponential backoff retry: 1s, 2s, 4s, 8s, max 30s
  │    └─ On reconnect: re-sync state with `get_agents`
  │
  └─ On app background:
       ├─ WebSocket stays open (foreground service if needed)
       └─ Go binary continues executing tasks
```

### 3.3 Streaming Protocol

For agent responses and status updates, the backend sends a stream of `agent_message` frames:

```
Backend sends:
  { "type": "agent_status",  "payload": { "status": "working", "detail": "Starting task..." } }
  { "type": "agent_status",  "payload": { "status": "working", "detail": "Reading file config.yaml..." } }
  { "type": "tool_execution","payload": { "tool": "read_file", "input": "config.yaml", "status": "running" } }
  { "type": "tool_execution","payload": { "tool": "read_file", "input": "config.yaml", "output": "...", "status": "done" } }
  { "type": "agent_message", "payload": { "content": "I've read the config file. Here's what I found:\n", "is_final": false } }
  { "type": "agent_message", "payload": { "content": "...the full analysis...", "is_final": true } }
  { "type": "agent_status",  "payload": { "status": "idle", "detail": "" } }
```

---

## 4. Agent State Machine

Each agent operates as an independent state machine managed by the Go backend.

### 4.1 States

```
                    ┌──────────┐
         ┌─────────│  CREATED  │
         │         └─────┬────┘
         │               │ (first message or task assigned)
         │               ▼
         │         ┌──────────┐
         │    ┌───►│   IDLE   │◄────────────────┐
         │    │    └─────┬────┘                  │
         │    │          │ user_message           │
         │    │          ▼                        │
         │    │    ┌───────────┐                  │
         │    │    │ THINKING  │──── LLM call ────┤
         │    │    └─────┬────┘                   │
         │    │          │ LLM responds with      │
         │    │          │ tool call or text       │
         │    │          ▼                        │
         │    │    ┌────────────┐                 │
         │    │    │ EXECUTING  │─── tool done ───┤
         │    │    │  (tools)   │                 │
         │    │    └─────┬─────┘                 │
         │    │          │                        │
         │    │          │ interrupt signal        │
         │    │          ▼                        │
         │    │    ┌──────────────┐               │
         │    │    │ INTERRUPTED  │───────────────┘
         │    │    └──────────────┘   (re-enters THINKING
         │    │                        with new context)
         │    │
         │    │    ┌──────────┐
         │    └────│  ERROR   │
         │         └──────────┘
         │               │ retry / dismiss
         │               │
         │         ┌──────────┐
         └────────►│ DELETED  │
                   └──────────┘
```

### 4.2 State Transitions

| From         | To           | Trigger                                 | Action                                                  |
|--------------|--------------|-----------------------------------------|---------------------------------------------------------|
| CREATED      | IDLE         | Agent initialization complete            | Register agent, broadcast `agent_list`                  |
| IDLE         | THINKING     | `user_message` received                  | Build prompt, call LLM                                  |
| THINKING     | EXECUTING    | LLM returns a tool call                  | Execute tool in sandboxed subprocess                    |
| THINKING     | IDLE         | LLM returns final text (no tool call)    | Send `agent_message` (is_final: true), update status    |
| EXECUTING    | THINKING     | Tool execution completes                 | Append tool result to context, re-call LLM              |
| EXECUTING    | ERROR        | Tool execution fails                     | Send error status, await user action                    |
| ANY_ACTIVE   | INTERRUPTED  | `interrupt` signal received              | Cancel current context, inject new message              |
| INTERRUPTED  | THINKING     | Immediately                              | Re-enter thinking loop with updated context             |
| ERROR        | IDLE         | `dismiss` received                       | Clear error state                                       |
| ERROR        | THINKING     | `retry` received                         | Re-attempt last action                                  |
| ANY          | DELETED      | `delete_agent` received                  | Cancel all work, clean up resources, remove from registry|

### 4.3 Agent Loop (Pseudocode)

```
func (a *Agent) Run(ctx context.Context) {
    for {
        select {
        case msg := <-a.incomingMessages:
            // New user message or interrupt
            a.context = append(a.context, msg)
            a.setState(THINKING)

            for {
                // Check for interrupts before each LLM call
                select {
                case interrupt := <-a.interruptChan:
                    a.handleInterrupt(interrupt)
                    continue outerLoop
                default:
                }

                response := a.llmClient.Call(ctx, a.context)

                if response.HasToolCall() {
                    a.setState(EXECUTING)
                    a.broadcast(toolExecutionStatus)

                    // Execute tool with cancellable context
                    toolCtx, cancel := context.WithCancel(ctx)
                    a.currentCancel = cancel

                    result := a.executeTool(toolCtx, response.ToolCall)
                    a.context = append(a.context, result)
                    a.setState(THINKING)
                } else {
                    // Final response
                    a.streamResponse(response.Text)
                    a.setState(IDLE)
                    break
                }
            }

        case <-ctx.Done():
            return
        }
    }
}
```

---

## 5. Interruptible Execution Model

This is the most critical architectural component. Agents must never ignore user input while working.

### 5.1 Mechanism

```
┌─────────────────────────────────────────────┐
│              Agent Goroutine                │
│                                             │
│   ┌───────────────┐    ┌────────────────┐   │
│   │ Execution Loop│    │ Interrupt Chan │   │
│   │               │◄───│   (buffered)   │   │
│   │  LLM Call     │    └────────┬───────┘   │
│   │    ↓          │             │           │
│   │  Tool Exec    │             │           │
│   │    ↓          │         User sends      │
│   │  LLM Call     │         new message     │
│   │    ...        │             │           │
│   └───────────────┘             │           │
│                                 │           │
│   Before each step:             │           │
│   select {                      │           │
│     case <-interruptChan:  ◄────┘           │
│       cancelCurrentWork()                   │
│       injectNewMessage()                    │
│       restartLoop()                         │
│     default:                                │
│       continueExecution()                   │
│   }                                         │
│                                             │
└─────────────────────────────────────────────┘
```

### 5.2 Interrupt Points

The agent checks for interrupts at these points (non-blocking select on the interrupt channel):

1. **Before each LLM API call** - cheapest interrupt point.
2. **Before each tool execution** - prevents starting work that will be discarded.
3. **During long tool execution** - via Go's `context.Context` cancellation. Shell commands are killed via process group signals (SIGTERM, then SIGKILL after timeout).
4. **During LLM streaming** - cancel the HTTP request mid-stream if interrupted.

### 5.3 Interrupt Flow (Detailed)

```
Timeline:
─────────────────────────────────────────────────────────

Agent:   [THINKING] ──► [EXECUTING: shell "npm install"] ──►
User:                              │
                          Sends "Actually, use yarn instead"
                                   │
                                   ▼
Backend: 1. Receives interrupt message via WebSocket
         2. Pushes message onto agent's interruptChan
         3. Calls agent.currentCancel() to cancel tool context
         4. Shell subprocess receives SIGTERM → killed
         5. Agent loop detects interrupt
         6. Broadcasts: { type: "interrupt_ack", ... }
         7. Appends to context:
            - "[System: Previous task interrupted by user]"
            - "[User: Actually, use yarn instead]"
         8. Re-enters THINKING state
         9. LLM sees full context including interruption
         10. Agent responds: "Got it, switching to yarn..."

Agent:   [INTERRUPTED] ──► [THINKING] ──► [EXECUTING: shell "yarn install"] ──►
```

### 5.4 Context Preservation on Interrupt

When an interrupt occurs, the agent's context is updated as follows:

```json
[
  { "role": "user", "content": "Set up a React project with npm" },
  { "role": "assistant", "content": "I'll set up a React project. Let me start by..." },
  { "role": "tool", "name": "shell", "content": "npm install [INTERRUPTED - killed by user]" },
  { "role": "system", "content": "The user interrupted the previous task." },
  { "role": "user", "content": "Actually, use yarn instead" }
]
```

This gives the LLM full awareness of what happened, enabling intelligent adaptation.

---

## 6. Multi-Agent Management

### 6.1 Agent Registry

The `AgentManager` maintains a concurrent-safe registry of all agents.

```
┌──────────────────────────────────────────┐
│            Agent Manager                 │
│                                          │
│  agents: map[string]*Agent               │
│    ├─ "uuid-1" → Sam (Main Agent)        │
│    ├─ "uuid-2" → Bob (Subagent of Sam)   │
│    └─ "uuid-3" → Alice (Subagent of Sam) │
│                                          │
│  Methods:                                │
│    CreateAgent(config) → Agent           │
│    DeleteAgent(id)                       │
│    GetAgent(id) → Agent                  │
│    ListAgents() → []Agent                │
│    RouteMessage(agentId, msg)            │
│    InterruptAgent(agentId, msg)          │
│    BroadcastStatus()                     │
│                                          │
└──────────────────────────────────────────┘
```

### 6.2 Agent Hierarchy

```
Main Agent (Sam)
  ├─ Subagent (Bob)
  │    └─ (can be assigned tasks by Sam or user directly)
  └─ Subagent (Alice)
       └─ (can be assigned tasks by Sam or user directly)
```

- **Main Agent:** The primary agent. Always exists. Can spawn subagents (in future phases).
- **Subagents:** Created by users (and later, by the main agent). Each has its own system prompt, tool access, and conversation context.
- **Independence:** Each agent has its own goroutine, context, state machine, and interrupt channel. They do NOT share memory or context unless explicitly designed to.

### 6.3 Context Switching

Context switching is purely a frontend concept. The backend always routes messages by `agent_id`.

```
Frontend:
  activeAgentId = "uuid-1" (Sam)
  
  User switches to Bob:
    activeAgentId = "uuid-2"
    Load Bob's message history (from local cache or request via WS)
    Subscribe to Bob's status updates
    
  Sam continues working in background:
    Status updates still arrive via WS (routed to Dashboard, not Chat)
    Sam's messages are buffered and displayed when user switches back
```

---

## 7. Tool Execution Framework

### 7.1 Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() JSONSchema
    Execute(ctx context.Context, input map[string]interface{}) (string, error)
}
```

### 7.2 Tool Registry

```go
type ToolRegistry struct {
    tools map[string]Tool
}

func (r *ToolRegistry) Register(tool Tool)
func (r *ToolRegistry) Get(name string) Tool
func (r *ToolRegistry) ListForAgent(agentConfig AgentConfig) []Tool
func (r *ToolRegistry) Execute(ctx context.Context, name string, input map[string]interface{}) (string, error)
```

### 7.3 Built-in Tools

#### `shell` - Shell Command Execution

| Property    | Value                                                    |
|-------------|----------------------------------------------------------|
| Name        | `shell`                                                  |
| Description | Execute a shell command on the device                    |
| Parameters  | `{ "command": string }`                                  |
| Execution   | Runs via `os/exec.CommandContext` with cancellable context |
| Output      | Combined stdout + stderr, truncated to 10KB              |
| Timeout     | Default 60s, configurable                                |
| Cancellation| Context cancellation sends SIGTERM to process group, SIGKILL after 5s grace period |
| Security    | Runs as app user. No root. No access outside app sandbox (Android-enforced). |

#### `read_file` - File Reading

| Property    | Value                                                    |
|-------------|----------------------------------------------------------|
| Name        | `read_file`                                              |
| Description | Read the contents of a file                              |
| Parameters  | `{ "path": string }`                                     |
| Execution   | `os.ReadFile` with context check before read              |
| Output      | File contents as string, truncated to 50KB                |
| Security    | Restricted to app's data directory and user-granted paths |

### 7.4 Tool Execution Flow

```
LLM Response: { "tool_calls": [{ "name": "shell", "arguments": { "command": "ls -la" } }] }
  │
  ├─ ToolRegistry.Get("shell") → ShellTool
  │
  ├─ Create cancellable context (linked to agent's interrupt)
  │    toolCtx, cancel := context.WithCancel(agentCtx)
  │
  ├─ Broadcast: { type: "tool_execution", tool: "shell", input: "ls -la", status: "running" }
  │
  ├─ ShellTool.Execute(toolCtx, { "command": "ls -la" })
  │    ├─ cmd := exec.CommandContext(toolCtx, "sh", "-c", "ls -la")
  │    ├─ cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}  // process group
  │    └─ output, err := cmd.CombinedOutput()
  │
  ├─ Broadcast: { type: "tool_execution", tool: "shell", output: "...", status: "done" }
  │
  └─ Return output to agent loop → append to context → next LLM call
```

---

## 8. LLM Integration Layer

### 8.1 Abstraction

```go
type LLMClient interface {
    // ChatCompletion sends messages and returns a streaming response
    ChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}

type ChatRequest struct {
    Model    string
    Messages []Message
    Tools    []ToolDefinition
    Stream   bool
}

type ChatChunk struct {
    Content   string      // text content (partial)
    ToolCalls []ToolCall  // tool call (if any)
    Done      bool        // is this the final chunk?
    Error     error       // error (if any)
}
```

### 8.2 Supported Backends

#### Cloud API Client (OpenAI-compatible)

- **Protocol:** HTTPS REST API
- **Endpoint:** User-configurable (e.g., `https://api.openai.com/v1`)
- **Auth:** Bearer token (API key)
- **Streaming:** SSE (Server-Sent Events) via `/chat/completions` with `stream: true`
- **Models:** Any model the endpoint supports
- **Compatibility:** Works with OpenAI, Azure OpenAI, Anthropic (via proxy), any OpenAI-compatible API

#### Ollama Client

- **Protocol:** HTTP REST API
- **Endpoint:** User-configurable (default `http://localhost:11434`)
- **Auth:** None (local)
- **Streaming:** NDJSON streaming via `/api/chat`
- **Models:** Any locally installed Ollama model

### 8.3 Endpoint Configuration

```json
{
  "endpoints": [
    {
      "id": "ep_uuid_1",
      "name": "GPT-4 Cloud",
      "type": "cloud",
      "url": "https://api.openai.com/v1",
      "api_key": "sk-...",
      "model": "gpt-4",
      "is_default": true
    },
    {
      "id": "ep_uuid_2",
      "name": "Local Llama",
      "type": "ollama",
      "url": "http://localhost:11434",
      "model": "llama3",
      "is_default": false
    }
  ]
}
```

---

## 9. Data Flow Diagrams

### 9.1 User Sends a Message (Happy Path)

```
 Flutter                    Go Backend                  LLM API
   │                           │                           │
   │  ── user_message ──►      │                           │
   │                           │                           │
   │  ◄── agent_status ──      │  (status: "thinking")     │
   │                           │                           │
   │                           │  ── ChatCompletion ──►    │
   │                           │                           │
   │                           │  ◄── stream chunk ──      │
   │  ◄── agent_message ──     │  (partial text)           │
   │                           │                           │
   │                           │  ◄── stream chunk ──      │
   │  ◄── agent_message ──     │  (tool_call: shell)       │
   │                           │                           │
   │  ◄── tool_execution ──    │  (shell: running)         │
   │                           │                           │
   │  ◄── agent_status ──      │  (status: "executing")    │
   │                           │                           │
   │                           │  [executes shell cmd]     │
   │                           │                           │
   │  ◄── tool_execution ──    │  (shell: done, output)    │
   │                           │                           │
   │                           │  ── ChatCompletion ──►    │
   │                           │  (with tool result)       │
   │                           │                           │
   │                           │  ◄── stream chunk ──      │
   │  ◄── agent_message ──     │  (final text)             │
   │                           │                           │
   │  ◄── agent_status ──      │  (status: "idle")         │
   │                           │                           │
```

### 9.2 User Interrupts an Agent

```
 Flutter                    Go Backend                  LLM API
   │                           │                           │
   │                           │  [EXECUTING: shell cmd]   │
   │                           │                           │
   │  ── interrupt ──►         │                           │
   │  { "Actually use yarn" }  │                           │
   │                           │                           │
   │                           │  cancel(toolCtx)          │
   │                           │  kill(shell process)      │
   │                           │                           │
   │  ◄── interrupt_ack ──     │                           │
   │                           │                           │
   │  ◄── agent_status ──      │  (status: "thinking")     │
   │                           │                           │
   │                           │  [rebuild context with    │
   │                           │   interrupt info]         │
   │                           │                           │
   │                           │  ── ChatCompletion ──►    │
   │                           │  (updated context)        │
   │                           │                           │
   │                           │  ◄── stream chunk ──      │
   │  ◄── agent_message ──     │  "Switching to yarn..."   │
   │                           │                           │
```

---

## 10. Security Model

### 10.1 Principles

| Principle               | Implementation                                                     |
|-------------------------|---------------------------------------------------------------------|
| **On-device only**      | Go backend binds to `127.0.0.1` only. No external network exposure. |
| **No root access**      | All tool execution runs under the Android app's UID.                |
| **Sandboxed file access**| `read_file` restricted to app data dir + user-granted directories.  |
| **API key encryption**  | API keys stored encrypted in SQLite using Android Keystore-derived key. |
| **Process isolation**   | Shell commands run in separate process groups for clean termination. |
| **Input sanitization**  | All WebSocket messages validated against JSON schema before processing. |
| **Resource limits**     | Tool outputs capped (shell: 10KB, file: 50KB). Shell timeout: 60s. |

### 10.2 Threat Mitigations

| Threat                    | Mitigation                                                        |
|---------------------------|-------------------------------------------------------------------|
| LLM prompt injection      | System prompts are clearly delineated. Tool outputs are marked as untrusted. User can review tool calls before execution (future phase). |
| Runaway shell commands     | Timeout + process group kill. Memory limits via Android OS.        |
| Data exfiltration          | Go backend only makes outbound requests to configured LLM endpoints. No other network calls. |
| WebSocket hijacking        | Bound to localhost. No authentication needed (same-device only).   |

---

## 11. Data Persistence

### 11.1 SQLite Schema (Core Tables)

```sql
-- Agents
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    system_prompt TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('main', 'sub')),
    parent_id TEXT REFERENCES agents(id),
    tools TEXT NOT NULL DEFAULT '[]',  -- JSON array of tool names
    llm_endpoint_id TEXT REFERENCES endpoints(id),
    status TEXT NOT NULL DEFAULT 'idle',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Conversations
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,
    tool_name TEXT,
    tool_input TEXT,
    is_interrupted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- LLM Endpoints
CREATE TABLE endpoints (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('cloud', 'ollama')),
    url TEXT NOT NULL,
    api_key TEXT,  -- encrypted
    model TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- App Settings
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Indexes
CREATE INDEX idx_messages_agent ON messages(agent_id, created_at);
CREATE INDEX idx_agents_parent ON agents(parent_id);
```

### 11.2 Data Flow

- **Write path:** Go backend writes to SQLite after each state change (message received, agent created, etc.)
- **Read path:** Frontend requests history via WebSocket; backend queries SQLite and returns paginated results.
- **Consistency:** Single-writer (Go backend). No concurrent write conflicts.

---

## 12. Error Handling Strategy

### 12.1 Error Categories

| Category          | Examples                                   | Handling                                                    |
|-------------------|--------------------------------------------|-------------------------------------------------------------|
| **LLM Errors**    | API timeout, rate limit, invalid key       | Retry with backoff (3 attempts). Then surface error to user via `agent_status` (error). |
| **Tool Errors**   | Command not found, permission denied        | Report error in tool result. LLM decides whether to retry or inform user. |
| **WebSocket Errors** | Connection drop, malformed message       | Auto-reconnect with backoff. Buffer unsent messages (up to 100). |
| **Backend Errors** | Panic, SQLite lock                         | Recover via goroutine recovery. Log error. Restart affected agent. |
| **Validation Errors** | Invalid agent config, missing fields    | Return `error` message with human-readable description.      |

### 12.2 Error Propagation

```
Tool Error → Agent Runtime → Status Broadcaster → WebSocket → Flutter → UI
                                                                         │
                                                          Error chip in chat
                                                          Error card on dashboard
                                                          Toast notification
```

### 12.3 Recovery Mechanisms

| Scenario                    | Recovery                                              |
|-----------------------------|-------------------------------------------------------|
| Agent stuck in EXECUTING    | 60s timeout → force kill → ERROR state                |
| LLM returns invalid JSON    | Parse error → retry LLM call with note about format   |
| WebSocket disconnect         | Flutter auto-reconnects + re-syncs state              |
| Go binary crashes            | Flutter detects disconnect → auto-restart binary       |
| SQLite corruption            | WAL mode for resilience. Export/import for recovery.   |
