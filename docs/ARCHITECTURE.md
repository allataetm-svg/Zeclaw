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
9. [Heartbeat System](#9-heartbeat-system)
10. [Cron / Scheduled Tasks](#10-cron--scheduled-tasks)
11. [Agent Memory & Persistence](#11-agent-memory--persistence)
12. [Skills System](#12-skills-system)
13. [Doctor / Diagnostics](#13-doctor--diagnostics)
14. [Hooks & Event-Driven Triggers](#14-hooks--event-driven-triggers)
15. [Agent Workspace](#15-agent-workspace)
16. [Audit Trail & Action Logging](#16-audit-trail--action-logging)
17. [Approval Gates](#17-approval-gates)
18. [Data Flow Diagrams](#18-data-flow-diagrams)
19. [Security Model](#19-security-model)
20. [Data Persistence (SQLite Schema)](#20-data-persistence-sqlite-schema)
21. [Error Handling Strategy](#21-error-handling-strategy)
22. [Agentic Feature Comparison Matrix](#22-agentic-feature-comparison-matrix)

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
| **Heartbeat Scheduler**| Runs periodic heartbeat ticks for each agent at configurable intervals. Triggers awareness checks. Inspired by OpenClaw's heartbeat system. |
| **Cron Scheduler**  | Manages recurring scheduled tasks using cron expressions or interval syntax. Fires agent tasks at specified times. Inspired by OpenClaw/Nanobot cron. |
| **Memory Manager**  | Handles agent long-term memory: store, search, forget, summarize. Persists to SQLite with semantic indexing. Inspired by OpenClaw/Nanobot memory systems. |
| **Skills Loader**   | Reads and injects SKILL.md files into agent context at runtime. Declarative capability definitions. Inspired by OpenClaw/OpenFang skill systems. |
| **Doctor Engine**   | Validates configuration, tests LLM provider connectivity, checks tool availability, reports system health. Inspired by OpenClaw/PicoClaw doctor command. |
| **Hook Dispatcher** | Manages event-driven triggers (file changes, task completion, timer expiry). Routes events to registered agent handlers. Inspired by OpenClaw hooks. |
| **Workspace Manager**| Per-agent file workspace on device for persistent artifacts, notes, and outputs. |
| **Audit Logger**    | Immutable append-only log of all agent actions (tool calls, LLM requests, state transitions). Inspired by OpenFang's Merkle audit trail. |
| **Approval Manager**| Intercepts sensitive tool calls (e.g., destructive shell commands) and requests user confirmation before execution. Inspired by OpenFang's approval gates. |

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
| `create_cron`         | `{ "agent_id": "...", "expression": "*/30 * * * *", "task": "...", "enabled": true }` | Schedule a recurring cron task for an agent. |
| `delete_cron`         | `{ "cron_id": "..." }`                               | Remove a scheduled cron task.            |
| `update_cron`         | `{ "cron_id": "...", "enabled": bool }`              | Enable/disable a cron task.              |
| `get_crons`           | `{ "agent_id": "..." }`                              | List cron tasks for an agent.            |
| `update_heartbeat`    | `{ "agent_id": "...", "enabled": bool, "interval": "30m" }` | Configure heartbeat for an agent.  |
| `memory_search`       | `{ "agent_id": "...", "query": "..." }`              | Search agent's long-term memory.         |
| `run_doctor`          | `{ "sections": ["config","providers","tools","system"] }` | Run diagnostics and health checks.  |
| `approve_action`      | `{ "approval_id": "...", "approved": bool }`         | Approve or deny a pending agent action.  |
| `get_audit_log`       | `{ "agent_id": "...", "limit": 50 }`                 | Fetch audit trail entries.               |
| `update_skill`        | `{ "agent_id": "...", "skill_name": "...", "content": "..." }` | Create or update a skill file.  |
| `get_skills`          | `{ "agent_id": "..." }`                              | List skills loaded for an agent.         |

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
| `heartbeat_result`    | `{ "agent_id": "...", "status": "ok|action", "summary": "...", "actions_taken": [...] }` | Result of a heartbeat tick. |
| `cron_triggered`      | `{ "cron_id": "...", "agent_id": "...", "task": "..." }` | Notification that a cron task fired.   |
| `cron_list`           | `{ "crons": [...] }`                                 | Response to `get_crons`.                 |
| `memory_results`      | `{ "memories": [...] }`                               | Response to `memory_search`.             |
| `doctor_report`       | `{ "sections": [...], "overall_status": "healthy|degraded|error" }` | Diagnostics report.        |
| `approval_request`    | `{ "approval_id": "...", "agent_id": "...", "tool": "...", "input": "...", "risk_level": "..." }` | Agent requests user approval for sensitive action. |
| `audit_log`           | `{ "entries": [...] }`                                | Response to `get_audit_log`.             |
| `skill_list`          | `{ "skills": [...] }`                                 | Response to `get_skills`.                |
| `hook_fired`          | `{ "hook_id": "...", "event": "...", "agent_id": "..." }` | Notification that a hook was triggered. |

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

## 9. Heartbeat System

> Inspired by: **OpenClaw** (heartbeat), **OpenFang** (scheduled Hands), **Nanobot** (heartbeat awareness)

The heartbeat is a periodic "awareness check" that runs inside each agent's session at a configurable interval. Unlike cron (which runs specific tasks at specific times), heartbeats let the agent proactively check on things and surface anything important — without the user having to ask.

### 9.1 Architecture

```
┌─────────────────────────────────────────────────────┐
│                Heartbeat Scheduler                  │
│                                                     │
│  Per-Agent Configuration:                           │
│    ├─ enabled: bool (default: false)                │
│    ├─ interval: duration (default: "30m")           │
│    ├─ activeHours: { start: "08:00", end: "22:00" } │
│    ├─ checklist: string (HEARTBEAT.md content)      │
│    └─ target: "chat" | "silent"                     │
│                                                     │
│  Scheduler Loop (per agent):                        │
│    ticker := time.NewTicker(interval)               │
│    for range ticker.C:                              │
│      if !inActiveHours(): continue                  │
│      if agent.state != IDLE: continue               │
│      agent.runHeartbeat(checklist)                   │
│                                                     │
│  Heartbeat Turn:                                    │
│    1. Inject heartbeat prompt into agent context     │
│    2. Agent reads HEARTBEAT.md checklist             │
│    3. Agent runs checks (cheap deterministic first)  │
│    4. If nothing to report → HEARTBEAT_OK (silent)   │
│    5. If action needed → surface to user via chat    │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### 9.2 HEARTBEAT.md (Per-Agent Checklist)

Each agent can have a `HEARTBEAT.md` stored in its workspace. The heartbeat prompt includes this file so the agent knows what to check.

```markdown
# Heartbeat Checklist
- Check if any background shell commands finished
- Look for new files in the workspace
- If any previous tasks errored, summarize in one sentence
- Check system resource usage (memory, storage)
```

### 9.3 Heartbeat Decision Pipeline

Following the pattern established by OpenClaw and OpenFang, the heartbeat uses a layered decision pipeline to minimize unnecessary LLM calls:

```
Heartbeat Tick
  │
  ├─ 1. Cheap deterministic checks (no LLM)
  │    ├─ Process status (any background jobs done?)
  │    ├─ File system changes (new files in workspace?)
  │    ├─ Error queue (any unresolved errors?)
  │    └─ Pending task status
  │
  ├─ 2. Rule evaluation
  │    ├─ Any threshold breached? (e.g., disk > 90%)
  │    └─ Any pending notifications?
  │
  ├─ 3. Escalation gate (decide if LLM is needed)
  │    ├─ If no issues → HEARTBEAT_OK (no LLM call, silent)
  │    └─ If issues found → escalate to LLM for analysis
  │
  └─ 4. Action dispatch
       ├─ Surface alert to user chat
       ├─ Auto-remediate (if configured)
       └─ Log to audit trail
```

### 9.4 Heartbeat Event Envelope

```json
{
  "agent_id": "uuid-1",
  "heartbeat_id": "hb_uuid",
  "timestamp": "2026-03-08T12:00:00Z",
  "checks": {
    "background_tasks": { "completed": 2, "failed": 0 },
    "workspace_changes": { "new_files": 1 },
    "error_queue": { "pending": 0 },
    "system": { "memory_mb": 45, "storage_free_mb": 2100 }
  },
  "escalation": {
    "llm_required": false,
    "reason": null
  },
  "result": "HEARTBEAT_OK"
}
```

---

## 10. Cron / Scheduled Tasks

> Inspired by: **OpenClaw** (cron system), **Nanobot** (scheduled tasks), **PicoClaw** (cron scheduling), **OpenFang** (scheduled Hands)

Cron lets users schedule recurring tasks for agents. Unlike heartbeats (which are awareness checks), cron runs specific user-defined tasks at exact times or intervals.

### 10.1 Cron Architecture

```
┌─────────────────────────────────────────────┐
│              Cron Scheduler                 │
│                                             │
│  CronEntry:                                 │
│    ├─ id: UUID                              │
│    ├─ agent_id: UUID                        │
│    ├─ expression: "*/30 * * * *"            │
│    │   OR interval: "every 2h"              │
│    ├─ task: "Check inbox and summarize"     │
│    ├─ enabled: bool                         │
│    ├─ run_mode: "main" | "isolated"         │
│    ├─ announce: bool (deliver to chat?)     │
│    ├─ created_at: timestamp                 │
│    └─ last_run: timestamp                   │
│                                             │
│  Scheduler:                                 │
│    - Parses cron expressions (5-field)      │
│    - Also supports interval syntax:         │
│      "every 15m", "every 2h", "every 1d"   │
│    - On tick: inject task as user_message   │
│      into agent's incoming queue            │
│    - Runs in main session (shared context)  │
│      or isolated session (fresh context)    │
│                                             │
└─────────────────────────────────────────────┘
```

### 10.2 Cron vs Heartbeat — When to Use Each

| Use Case | Recommended | Why |
|----------|-------------|-----|
| Check if background tasks finished | Heartbeat | Batches with other checks, context-aware |
| Send daily summary at 9am | Cron (isolated) | Exact timing needed |
| Monitor workspace for changes | Heartbeat | Natural fit for periodic awareness |
| Run weekly code cleanup | Cron (isolated) | Standalone task, different context needed |
| Remind user in 20 minutes | Cron (one-shot) | Precise timing, one-time |
| Check system resource usage | Heartbeat | Low-overhead, piggybacks on cycle |

### 10.3 Cron Execution Modes

- **Main session** (`run_mode: "main"`): Task runs in the agent's main conversation context. The agent remembers recent conversations and can follow up naturally. Good for tasks that relate to ongoing work.
- **Isolated session** (`run_mode: "isolated"`): Task runs in a fresh context with only the system prompt and task instruction. Good for standalone tasks that shouldn't pollute conversation history.

### 10.4 One-Shot Cron (Reminders)

Cron also supports one-shot tasks (run once at a specific time, then auto-delete):

```json
{
  "id": "cron_uuid",
  "agent_id": "uuid-1",
  "expression": null,
  "run_at": "2026-03-08T14:30:00Z",
  "task": "Remind the user about the team meeting",
  "one_shot": true,
  "announce": true
}
```

---

## 11. Agent Memory & Persistence

> Inspired by: **OpenClaw** (MEMORY.md, SESSION-STATE.md, memory-tools), **Nanobot** (long-term memory, daily notes), **OpenFang** (knowledge graphs)

Agents need memory that survives beyond a single conversation session. The memory system has three layers.

### 11.1 Memory Architecture

```
┌───────────────────────────────────────────────────┐
│                Memory Layers                      │
│                                                   │
│  Layer 1: Session Context (Volatile)              │
│    ├─ Current conversation messages               │
│    ├─ Active tool results                         │
│    └─ Lives in agent goroutine memory             │
│    └─ Lost on agent restart                       │
│                                                   │
│  Layer 2: Long-Term Memory (Persistent)           │
│    ├─ Stored in SQLite `memories` table           │
│    ├─ Agent decides WHEN to store (not auto)      │
│    ├─ Supports: store, search, forget, summarize  │
│    ├─ Each memory has:                            │
│    │    ├─ content: string                        │
│    │    ├─ category: "fact" | "preference" |      │
│    │    │            "instruction" | "learned"     │
│    │    ├─ importance: float (0.0 - 1.0)          │
│    │    ├─ confidence: float (0.0 - 1.0)          │
│    │    └─ created_at / updated_at                │
│    └─ Keyword search + optional semantic search   │
│                                                   │
│  Layer 3: Workspace Files (Persistent)            │
│    ├─ MEMORY.md — Free-form notes the agent keeps │
│    ├─ SESSION-STATE.md — Hot context for recovery  │
│    └─ Files in agent's workspace directory        │
│                                                   │
└───────────────────────────────────────────────────┘
```

### 11.2 Memory Tools (Available to LLM)

The memory system is exposed to the agent as internal tools that the LLM can invoke:

| Tool | Description | Parameters |
|------|-------------|------------|
| `memory_store` | Save a memory for later recall | `{ "content": "...", "category": "...", "importance": 0.8 }` |
| `memory_search` | Search memories by keyword | `{ "query": "...", "limit": 10 }` |
| `memory_forget` | Remove a specific memory | `{ "memory_id": "..." }` |
| `memory_summarize` | Condense old memories into summaries | `{ "older_than": "7d" }` |
| `memory_list` | List recent memories | `{ "category": "...", "limit": 20 }` |

### 11.3 Memory Injection into Context

Before each LLM call, relevant memories are injected into the system prompt:

```
[System Prompt]
[Agent's SKILL.md files]
[Relevant memories from search (top 5 by importance)]
[HEARTBEAT.md if heartbeat turn]
[Conversation messages...]
```

### 11.4 Memory Decay

Memories that haven't been accessed in 30 days have their importance score reduced by 10% per week. Memories with importance below 0.1 are candidates for auto-summarization or deletion (configurable).

---

## 12. Skills System

> Inspired by: **OpenClaw** (SKILL.md, ClawHub skill marketplace), **OpenFang** (SKILL.md + HAND.toml), **Nanobot** (skill extension)

Skills are declarative capability definitions that teach agents specialized behaviors. Each skill is a markdown file with YAML frontmatter that gets injected into the agent's context.

### 12.1 Skill Structure

```
workspace/skills/
  ├── coding-assistant.md
  ├── research-helper.md
  └── devops-monitor.md
```

Each skill file follows this format:

```markdown
---
name: coding-assistant
description: Expert coding assistant with best practices
tools_required: [shell, read_file]
---

# Coding Assistant

## When to Use
Use this skill when the user asks for help writing, reviewing, or debugging code.

## Behavior Rules
1. Always check existing code style before writing new code
2. Write tests for any new functionality
3. Use shell tool to verify code compiles before presenting it

## Response Format
- Start with a brief analysis of the problem
- Present solution with code blocks
- Explain key decisions
```

### 12.2 Skill Loading

```
Agent Initialization
  │
  ├─ Load built-in skills from app bundle
  │    └─ Default skills: general-assistant, coding-helper
  │
  ├─ Load workspace skills from agent's workspace/skills/
  │    └─ User-created skills specific to this agent
  │
  ├─ Filter by agent's tool access
  │    └─ Skip skills requiring tools the agent doesn't have
  │
  └─ Inject into system prompt (before conversation)
       └─ Skills appear after system prompt, before memories
```

### 12.3 Skill Precedence

1. **Agent system prompt** (highest) — Always takes priority
2. **Workspace skills** — User-created, per-agent
3. **Built-in skills** — Bundled defaults (lowest)

---

## 13. Doctor / Diagnostics

> Inspired by: **OpenClaw** (`openclaw doctor`), **PicoClaw** (proposed `picoclaw doctor`)

The Doctor system validates configuration, checks provider health, and reports system diagnostics. It's the first thing to run when something isn't working.

### 13.1 Doctor Checks

```
Doctor Report
  │
  ├─ 1. Configuration Validation
  │    ├─ Check all required settings are present
  │    ├─ Validate LLM endpoint URLs are well-formed
  │    ├─ Verify API keys are non-empty (not decrypted — just present)
  │    └─ Check agent configs have valid system prompts
  │
  ├─ 2. Provider Health Checks
  │    ├─ For each configured LLM endpoint:
  │    │    ├─ HTTP connectivity test (with 10s timeout)
  │    │    ├─ Auth validation (send minimal request)
  │    │    ├─ Model availability check
  │    │    └─ Latency measurement
  │    └─ Ollama: check if running + list models
  │
  ├─ 3. Tool Health Checks
  │    ├─ Shell: verify `sh` is available
  │    ├─ Read file: verify workspace directory exists
  │    └─ Check tool permissions
  │
  ├─ 4. System Health
  │    ├─ Go backend status (process alive, WebSocket listening)
  │    ├─ SQLite database integrity check
  │    ├─ Available storage space
  │    ├─ Memory usage
  │    └─ WebSocket connection state
  │
  └─ Output: doctor_report message to frontend
```

### 13.2 Doctor Report Format

```json
{
  "overall_status": "degraded",
  "timestamp": "2026-03-08T12:00:00Z",
  "sections": [
    {
      "name": "configuration",
      "status": "healthy",
      "checks": [
        { "name": "settings_present", "pass": true },
        { "name": "endpoint_urls_valid", "pass": true }
      ]
    },
    {
      "name": "providers",
      "status": "degraded",
      "checks": [
        { "name": "GPT-4 Cloud", "pass": true, "latency_ms": 230 },
        { "name": "Local Ollama", "pass": false, "error": "connection refused" }
      ]
    },
    {
      "name": "tools",
      "status": "healthy",
      "checks": [
        { "name": "shell", "pass": true },
        { "name": "read_file", "pass": true }
      ]
    },
    {
      "name": "system",
      "status": "healthy",
      "checks": [
        { "name": "backend_alive", "pass": true },
        { "name": "sqlite_integrity", "pass": true },
        { "name": "storage_free_mb", "pass": true, "value": 2100 }
      ]
    }
  ],
  "suggestions": [
    "Ollama is not reachable at localhost:11434. Start Ollama or update the endpoint in Settings."
  ]
}
```

### 13.3 Design Safety Note

Unlike OpenClaw's `doctor --fix` (which has a known issue of mutating/deleting config fields), Zeclaw's doctor is **read-only by default**. It outputs suggestions as actionable items the user can apply, but does not auto-modify configuration. This avoids the risk of destructive auto-fixes.

---

## 14. Hooks & Event-Driven Triggers

> Inspired by: **OpenClaw** (hooks system — "when something happens, do this")

Hooks allow agents to react to events without user prompting. They bridge the gap between fully manual operation and fully autonomous behavior.

### 14.1 Hook Types

| Hook Type | Trigger | Example |
|-----------|---------|---------|
| `on_task_complete` | Agent finishes a task | Notify user, run follow-up |
| `on_task_error` | Agent encounters an error | Auto-retry, alert user |
| `on_file_change` | File modified in workspace | Re-analyze, update summary |
| `on_cron_fire` | Scheduled cron task triggers | Execute the cron task |
| `on_agent_idle` | Agent transitions to IDLE | Check for queued work |
| `on_heartbeat` | Heartbeat tick fires | Run awareness checks |

### 14.2 Hook Configuration

```json
{
  "hooks": [
    {
      "id": "hook_uuid",
      "agent_id": "uuid-1",
      "event": "on_task_complete",
      "action": "message",
      "config": {
        "content": "Task finished. Run `doctor` to check system health."
      }
    },
    {
      "id": "hook_uuid_2",
      "agent_id": "uuid-1",
      "event": "on_task_error",
      "action": "retry",
      "config": {
        "max_retries": 2,
        "backoff_seconds": 10
      }
    }
  ]
}
```

### 14.3 Hook Execution

```
Event occurs (e.g., agent finishes task)
  │
  ├─ Hook Dispatcher checks registered hooks for this event
  │
  ├─ For each matching hook:
  │    ├─ Validate hook is enabled
  │    ├─ Execute action:
  │    │    ├─ "message" → inject message into agent's incoming queue
  │    │    ├─ "retry" → re-run last failed action
  │    │    ├─ "notify" → send notification to frontend
  │    │    └─ "agent_task" → send a new task to the agent
  │    └─ Log hook execution to audit trail
  │
  └─ Broadcast hook_fired event to frontend
```

---

## 15. Agent Workspace

> Inspired by: **OpenClaw** (persistent workspace at `~/.openclaw/workspace/`), **OpenFang** (per-Hand workspace), **Nanobot** (workspace)

Each agent gets a dedicated directory on the Android device's internal storage for persistent files, notes, and artifacts.

### 15.1 Workspace Layout

```
/data/data/com.zeclaw.app/files/workspaces/
  ├── {agent-uuid-1}/               # Sam's workspace
  │    ├── MEMORY.md                 # Agent's free-form notes
  │    ├── HEARTBEAT.md              # Heartbeat checklist
  │    ├── SESSION-STATE.md          # Hot context for session recovery
  │    ├── skills/                   # Per-agent skill files
  │    │    └── coding-assistant.md
  │    ├── outputs/                  # Files created by agent
  │    │    ├── report.md
  │    │    └── analysis.json
  │    └── temp/                     # Ephemeral working directory
  │
  ├── {agent-uuid-2}/               # Bob's workspace
  │    ├── MEMORY.md
  │    └── ...
  │
  └── shared/                        # Shared across all agents
       ├── AGENTS.md                  # Cross-agent coordination notes
       └── knowledge/                 # Shared knowledge base
```

### 15.2 Workspace Access Rules

- Each agent can only read/write within its own workspace by default
- The `shared/` directory is readable by all agents
- The `read_file` tool is restricted to workspace paths + user-granted paths
- Shell tool commands run with the agent's workspace as the working directory

---

## 16. Audit Trail & Action Logging

> Inspired by: **OpenFang** (Merkle audit trail, 16 security layers), **OpenClaw** (action logging)

Every significant agent action is recorded in an immutable, append-only audit log. This provides transparency, debugging capability, and accountability.

### 16.1 What Gets Logged

| Event Type | Data Recorded |
|------------|---------------|
| `tool_call` | Tool name, input, output (truncated), duration, success/failure |
| `llm_request` | Model, token count (input/output), latency, cost estimate |
| `state_change` | Agent ID, from_state, to_state, trigger |
| `interrupt` | Agent ID, interrupted task, new message |
| `memory_write` | Memory ID, category, content preview |
| `cron_fire` | Cron ID, task, result |
| `heartbeat` | Agent ID, checks summary, escalation decision |
| `approval_request` | Action details, user decision, latency |
| `hook_fire` | Hook ID, event, action taken |
| `error` | Category, message, stack trace (if applicable) |

### 16.2 Audit Entry Schema

```json
{
  "id": "audit_uuid",
  "timestamp": "2026-03-08T12:00:00Z",
  "agent_id": "uuid-1",
  "event_type": "tool_call",
  "data": {
    "tool": "shell",
    "input": "npm install express",
    "output_preview": "added 57 packages...",
    "duration_ms": 3200,
    "success": true
  },
  "session_id": "session_uuid"
}
```

### 16.3 Audit Storage

- Stored in SQLite `audit_log` table (append-only)
- Retained for 30 days by default (configurable)
- Older entries can be exported to JSON before pruning
- Frontend can view audit log from Settings > View Logs or per-agent from Dashboard

---

## 17. Approval Gates

> Inspired by: **OpenFang** (approval manager, guardrails on Hands), **OpenClaw** (permission gates)

For sensitive or potentially destructive operations, agents must request user approval before proceeding. This prevents agents from accidentally deleting files, running dangerous commands, or making irreversible changes.

### 17.1 Risk Classification

| Risk Level | Examples | Behavior |
|------------|----------|----------|
| **Low** | `read_file`, `ls`, `echo` | Execute immediately, no approval needed |
| **Medium** | `npm install`, `mkdir`, write to workspace | Execute immediately, log to audit trail |
| **High** | `rm`, `sudo`, `kill`, write outside workspace | **Pause and request user approval** |
| **Critical** | Network requests, system config changes | **Block until explicit user approval** |

### 17.2 Approval Flow

```
Agent wants to execute: rm -rf node_modules/
  │
  ├─ Tool Executor classifies risk → HIGH
  │
  ├─ Agent state → WAITING_APPROVAL (new sub-state of EXECUTING)
  │
  ├─ Backend sends approval_request to frontend:
  │    {
  │      "approval_id": "appr_uuid",
  │      "agent_id": "uuid-1",
  │      "tool": "shell",
  │      "input": "rm -rf node_modules/",
  │      "risk_level": "high",
  │      "reason": "Destructive file operation detected"
  │    }
  │
  ├─ Frontend shows approval dialog:
  │    ┌────────────────────────────────────┐
  │    │  ⚠️ Agent Sam requests approval    │
  │    │                                    │
  │    │  Action: shell                     │
  │    │  Command: rm -rf node_modules/     │
  │    │  Risk: HIGH                        │
  │    │                                    │
  │    │  [ Deny ]           [ Approve ]    │
  │    └────────────────────────────────────┘
  │
  ├─ User responds → approve_action message sent to backend
  │
  └─ If approved: execute tool, continue agent loop
     If denied: inject denial into context, agent adapts
```

### 17.3 Risk Detection Rules

Risk classification uses pattern matching on tool inputs:

```go
var highRiskPatterns = []string{
    `rm\s+(-rf?|-r)\s+`,     // recursive delete
    `sudo\s+`,                // privilege escalation
    `kill\s+`,                // process killing
    `chmod\s+`,               // permission changes
    `>\s*/`,                   // overwrite system files
    `mkfs`,                    // format filesystem
    `dd\s+`,                   // disk operations
}
```

Users can customize risk rules in Settings (future phase).

---

## 18. Data Flow Diagrams

### 18.1 User Sends a Message (Happy Path)

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

### 18.2 User Interrupts an Agent

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

## 19. Security Model

### 19.1 Principles

| Principle               | Implementation                                                     |
|-------------------------|---------------------------------------------------------------------|
| **On-device only**      | Go backend binds to `127.0.0.1` only. No external network exposure. |
| **No root access**      | All tool execution runs under the Android app's UID.                |
| **Sandboxed file access**| `read_file` restricted to app data dir + user-granted directories.  |
| **API key encryption**  | API keys stored encrypted in SQLite using Android Keystore-derived key. |
| **Process isolation**   | Shell commands run in separate process groups for clean termination. |
| **Input sanitization**  | All WebSocket messages validated against JSON schema before processing. |
| **Resource limits**     | Tool outputs capped (shell: 10KB, file: 50KB). Shell timeout: 60s. |

### 19.2 Threat Mitigations

| Threat                    | Mitigation                                                        |
|---------------------------|-------------------------------------------------------------------|
| LLM prompt injection      | System prompts are clearly delineated. Tool outputs are marked as untrusted. User can review tool calls before execution (future phase). |
| Runaway shell commands     | Timeout + process group kill. Memory limits via Android OS.        |
| Data exfiltration          | Go backend only makes outbound requests to configured LLM endpoints. No other network calls. |
| WebSocket hijacking        | Bound to localhost. No authentication needed (same-device only).   |

---

## 20. Data Persistence

### 20.1 SQLite Schema (Core Tables)

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

### 20.2 SQLite Schema (Agentic Feature Tables)

```sql
-- Agent Memories (long-term memory system)
CREATE TABLE memories (
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

-- Cron Scheduled Tasks
CREATE TABLE cron_tasks (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    expression TEXT,                    -- cron expression (5-field)
    interval TEXT,                      -- interval syntax ("every 2h")
    run_at TIMESTAMP,                   -- one-shot timestamp
    task TEXT NOT NULL,                 -- task description / prompt
    run_mode TEXT NOT NULL DEFAULT 'main' CHECK(run_mode IN ('main', 'isolated')),
    announce BOOLEAN DEFAULT TRUE,
    one_shot BOOLEAN DEFAULT FALSE,
    enabled BOOLEAN DEFAULT TRUE,
    last_run TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Heartbeat Configuration (per agent)
CREATE TABLE heartbeat_config (
    agent_id TEXT PRIMARY KEY REFERENCES agents(id),
    enabled BOOLEAN DEFAULT FALSE,
    interval_seconds INTEGER DEFAULT 1800,  -- 30 minutes
    active_hours_start TEXT DEFAULT '08:00',
    active_hours_end TEXT DEFAULT '22:00',
    target TEXT DEFAULT 'chat' CHECK(target IN ('chat', 'silent')),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Hooks (event-driven triggers)
CREATE TABLE hooks (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    event TEXT NOT NULL,                -- on_task_complete, on_task_error, etc.
    action TEXT NOT NULL,               -- message, retry, notify, agent_task
    config TEXT NOT NULL DEFAULT '{}',  -- JSON config for the action
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit Log (append-only)
CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    agent_id TEXT REFERENCES agents(id),
    session_id TEXT,
    event_type TEXT NOT NULL,           -- tool_call, llm_request, state_change, etc.
    data TEXT NOT NULL DEFAULT '{}',    -- JSON event data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Approval Requests
CREATE TABLE approval_requests (
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

-- Skills (per agent)
CREATE TABLE skills (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    name TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,              -- full skill markdown content
    tools_required TEXT DEFAULT '[]',   -- JSON array of tool names
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for new tables
CREATE INDEX idx_memories_agent ON memories(agent_id, category);
CREATE INDEX idx_memories_importance ON memories(agent_id, importance DESC);
CREATE INDEX idx_cron_agent ON cron_tasks(agent_id, enabled);
CREATE INDEX idx_audit_agent ON audit_log(agent_id, created_at);
CREATE INDEX idx_audit_type ON audit_log(event_type, created_at);
CREATE INDEX idx_approval_status ON approval_requests(status, created_at);
CREATE INDEX idx_skills_agent ON skills(agent_id, enabled);
```

### 20.3 Data Flow

- **Write path:** Go backend writes to SQLite after each state change (message received, agent created, etc.)
- **Read path:** Frontend requests history via WebSocket; backend queries SQLite and returns paginated results.
- **Consistency:** Single-writer (Go backend). No concurrent write conflicts.

---

## 21. Error Handling Strategy

### 21.1 Error Categories

| Category          | Examples                                   | Handling                                                    |
|-------------------|--------------------------------------------|-------------------------------------------------------------|
| **LLM Errors**    | API timeout, rate limit, invalid key       | Retry with backoff (3 attempts). Then surface error to user via `agent_status` (error). |
| **Tool Errors**   | Command not found, permission denied        | Report error in tool result. LLM decides whether to retry or inform user. |
| **WebSocket Errors** | Connection drop, malformed message       | Auto-reconnect with backoff. Buffer unsent messages (up to 100). |
| **Backend Errors** | Panic, SQLite lock                         | Recover via goroutine recovery. Log error. Restart affected agent. |
| **Validation Errors** | Invalid agent config, missing fields    | Return `error` message with human-readable description.      |

### 21.2 Error Propagation

```
Tool Error → Agent Runtime → Status Broadcaster → WebSocket → Flutter → UI
                                                                         │
                                                          Error chip in chat
                                                          Error card on dashboard
                                                          Toast notification
```

### 21.3 Recovery Mechanisms

| Scenario                    | Recovery                                              |
|-----------------------------|-------------------------------------------------------|
| Agent stuck in EXECUTING    | 60s timeout → force kill → ERROR state                |
| LLM returns invalid JSON    | Parse error → retry LLM call with note about format   |
| WebSocket disconnect         | Flutter auto-reconnects + re-syncs state              |
| Go binary crashes            | Flutter detects disconnect → auto-restart binary       |
| SQLite corruption            | WAL mode for resilience. Export/import for recovery.   |

---

## 22. Agentic Feature Comparison Matrix

The following matrix maps features discovered across OpenClaw and its alternatives to Zeclaw's implementation plan. This shows the origin/inspiration for each feature and which implementation phase it targets.

| Feature | OpenClaw | Nanobot | OpenFang | PicoClaw | NullClaw | ZeroClaw | TinyClaw | IronClaw | NanoClaw | **Zeclaw** | **Phase** |
|---------|----------|---------|----------|----------|----------|----------|----------|----------|----------|-----------|----------|
| **Heartbeat** | Core | Core | Via Hands | Proposed | - | - | Lite | - | - | **Yes** | Phase 7 |
| **Cron / Scheduled Tasks** | Core | Core | Via Hands | Core | Basic | Basic | - | Core | Basic | **Yes** | Phase 7 |
| **Doctor / Diagnostics** | Core | - | - | Proposed | - | - | - | Health API | - | **Yes** | Phase 6 |
| **Skills / SKILL.md** | Core | Extension | Core | - | - | - | - | Plugin | - | **Yes** | Phase 8 |
| **Long-Term Memory** | Core | Core | - | - | - | Basic | - | - | Basic | **Yes** | Phase 7 |
| **Hooks / Events** | Core | Partial | Events | - | - | - | - | Webhook | - | **Yes** | Phase 8 |
| **Workspace (per-agent files)** | Core | Core | Per-Hand | - | - | - | - | - | - | **Yes** | Phase 7 |
| **Audit Trail** | Logging | - | Merkle Trail | - | - | - | - | Logging | - | **Yes** | Phase 8 |
| **Approval Gates** | Future | - | Core | - | - | - | - | - | - | **Yes** | Phase 6 |
| **Multi-Agent** | Core | - | Core (Hands) | - | - | - | - | - | - | **Yes** | Phase 6 |
| **Interruptible Execution** | - | - | - | - | - | - | - | - | - | **Yes (Core)** | Phase 5 |
| **Real-Time Streaming** | Partial | Partial | - | - | - | - | - | - | - | **Yes (Core)** | Phase 4 |
| **Knowledge Graphs** | - | - | Core | - | - | - | - | - | - | **Future** | TBD |
| **Multi-Channel (40+)** | - | - | Core | - | - | - | - | - | - | **N/A** | Mobile-only |
| **WASM Sandboxing** | - | - | Core | - | - | - | - | - | - | **Future** | TBD |
| **Hands (24/7 Workers)** | - | - | Core | - | - | - | - | - | - | **Future** | TBD |
| **MCP Support** | Partial | - | Core | - | - | - | - | - | - | **Future** | TBD |
| **AGENTS.md (Behavior Def)** | Core | - | - | - | - | - | - | - | - | **Yes** | Phase 8 |
| **Memory Decay / Confidence** | Advanced | - | - | - | - | - | - | - | - | **Yes** | Phase 7 |

### Key Takeaways

- **Core differentiator**: Zeclaw's interruptible execution model is unique — no competitor implements true mid-task interruption with context adaptation.
- **Feature parity targets**: Heartbeat, Cron, Doctor, Skills, Memory, and Approval Gates bring Zeclaw to feature parity with OpenClaw/OpenFang for the features that make sense on mobile.
- **Deferred features**: Knowledge Graphs, WASM Sandboxing, Hands (24/7 workers), Multi-Channel, and MCP Support are documented for future phases but not critical for mobile-first launch.
- **Mobile adaptation**: All features are adapted for Android constraints (battery-aware heartbeat, configurable active hours, efficient SQLite storage, workspace scoped to app directory).
