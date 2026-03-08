# Zeclaw - Step-by-Step Implementation Plan

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 0: Environment & Project Setup](#phase-0-environment--project-setup)
3. [Phase 1: Go Backend Core](#phase-1-go-backend-core)
4. [Phase 2: Flutter Frontend Shell](#phase-2-flutter-frontend-shell)
5. [Phase 3: Agent Engine (Single Agent)](#phase-3-agent-engine-single-agent)
6. [Phase 4: Real-Time Streaming & WebSocket Integration](#phase-4-real-time-streaming--websocket-integration)
7. [Phase 5: Interruptible Execution](#phase-5-interruptible-execution)
8. [Phase 6: Multi-Agent System](#phase-6-multi-agent-system)
9. [Phase 7: Dashboard & Monitoring](#phase-7-dashboard--monitoring)
10. [Phase 8: Settings & LLM Configuration](#phase-8-settings--llm-configuration)
11. [Phase 9: Polish, Testing & Hardening](#phase-9-polish-testing--hardening)
12. [Phase 10: Packaging & Distribution](#phase-10-packaging--distribution)
13. [Phase 11: Doctor & Approval Gates](#phase-11-doctor--approval-gates)
14. [Phase 12: Heartbeat, Cron & Memory](#phase-12-heartbeat-cron--memory)
15. [Phase 13: Skills, Hooks & Workspace](#phase-13-skills-hooks--workspace)
16. [Phase 14: Audit Trail & Agent Behavior](#phase-14-audit-trail--agent-behavior)
17. [Dependency Map](#dependency-map)
18. [Risk Register](#risk-register)

---

## Overview

### Build Philosophy

- **Vertical slices:** Each phase delivers a working (if minimal) end-to-end feature.
- **Backend-first:** The Go backend is built and tested before the Flutter UI consumes it.
- **Incremental complexity:** Start with one agent, one tool, one LLM. Then layer in multi-agent, interrupts, and richer UI.

### Estimated Timeline

| Phase | Name                              | Estimated Duration |
|-------|-----------------------------------|--------------------|
| 0     | Environment & Project Setup       | 1-2 days           |
| 1     | Go Backend Core                   | 3-4 days           |
| 2     | Flutter Frontend Shell             | 2-3 days           |
| 3     | Agent Engine (Single Agent)        | 4-5 days           |
| 4     | Real-Time Streaming & WebSocket    | 3-4 days           |
| 5     | Interruptible Execution            | 3-4 days           |
| 6     | Multi-Agent System                 | 3-4 days           |
| 7     | Dashboard & Monitoring             | 2-3 days           |
| 8     | Settings & LLM Configuration       | 2-3 days           |
| 9     | Polish, Testing & Hardening        | 4-5 days           |
| 10    | Packaging & Distribution           | 2-3 days           |
| 11    | Doctor & Approval Gates            | 3-4 days           |
| 12    | Heartbeat, Cron & Memory           | 5-7 days           |
| 13    | Skills, Hooks & Workspace          | 4-5 days           |
| 14    | Audit Trail & Agent Behavior       | 3-4 days           |
|       | **Total**                          | **~45-55 days**    |

---

## Phase 0: Environment & Project Setup

### Objective
Set up the development environment, project structure, and CI/CD pipeline.

### Steps

#### 0.1 Development Environment
- [ ] Install Flutter SDK (stable channel, latest version)
- [ ] Install Go (1.22+)
- [ ] Install Android Studio + Android SDK (API 33+)
- [ ] Install Android NDK (for cross-compiling Go to ARM)
- [ ] Set up an Android emulator (Pixel 7 API 33 recommended)
- [ ] Install `gomobile` for Go-Android integration
- [ ] Verify `flutter doctor` passes all checks

#### 0.2 Repository Structure
- [ ] Initialize the monorepo structure:

```
zeclaw/
├── docs/                        # Documentation (this folder)
│   ├── ARCHITECTURE.md
│   ├── IMPLEMENTATION_PLAN.md
│   └── APP_UI_UX_VIEW.md
├── frontend/                    # Flutter application
│   ├── lib/
│   │   ├── main.dart
│   │   ├── app/                 # App-level config (theme, router)
│   │   ├── features/            # Feature modules
│   │   │   ├── chat/
│   │   │   ├── dashboard/
│   │   │   └── settings/
│   │   ├── core/                # Shared utilities
│   │   │   ├── websocket/       # WebSocket client
│   │   │   ├── models/          # Data models
│   │   │   ├── providers/       # Riverpod providers
│   │   │   └── theme/           # Dark theme definition
│   │   └── widgets/             # Reusable components
│   ├── pubspec.yaml
│   └── test/
├── backend/                     # Go backend binary
│   ├── cmd/
│   │   └── zeclaw/
│   │       └── main.go          # Entry point
│   ├── internal/
│   │   ├── server/              # WebSocket server
│   │   ├── agent/               # Agent engine & state machine
│   │   ├── tools/               # Tool executors
│   │   ├── llm/                 # LLM client abstraction
│   │   ├── storage/             # SQLite persistence
│   │   └── protocol/            # Message types & serialization
│   ├── go.mod
│   ├── go.sum
│   └── Makefile
├── scripts/                     # Build & utility scripts
│   ├── build-backend.sh         # Cross-compile Go for Android
│   └── run-dev.sh               # Start dev environment
├── .github/
│   └── workflows/
│       └── ci.yml               # CI pipeline
└── README.md
```

- [ ] Initialize Git repository with `.gitignore` for Flutter, Go, Android artifacts
- [ ] Create `README.md` with project overview and setup instructions

#### 0.3 CI/CD Pipeline
- [ ] Set up GitHub Actions workflow:
  - Go: `go vet`, `go test`, `golangci-lint`
  - Flutter: `flutter analyze`, `flutter test`
  - Build check: ensure both projects compile without errors

#### 0.4 Verification
- [ ] `flutter create` frontend app and run on emulator (blank screen is fine)
- [ ] `go mod init` backend and run a "Hello World" HTTP server
- [ ] Confirm both compile and run independently

---

## Phase 1: Go Backend Core

### Objective
Build the foundational Go backend: WebSocket server, message protocol, and SQLite persistence.

### Steps

#### 1.1 Project Initialization
- [ ] Set up `go.mod` with module path `github.com/allataetm-svg/zeclaw/backend`
- [ ] Add dependencies:
  - `github.com/gorilla/websocket` - WebSocket server
  - `modernc.org/sqlite` - Pure Go SQLite (no CGO, critical for Android cross-compilation)
  - `github.com/google/uuid` - UUID generation

#### 1.2 Message Protocol
- [ ] Define message envelope struct in `internal/protocol/`:

```go
type Envelope struct {
    ID        string          `json:"id"`
    Type      string          `json:"type"`
    AgentID   string          `json:"agent_id"`
    Timestamp time.Time       `json:"timestamp"`
    Payload   json.RawMessage `json:"payload"`
}
```

- [ ] Define all message types (see Architecture doc Section 3.1):
  - Inbound: `user_message`, `interrupt`, `create_agent`, `delete_agent`, `stop_agent`, `retry_agent`, `update_settings`, `get_agents`, `get_history`
  - Outbound: `agent_message`, `agent_status`, `tool_execution`, `agent_list`, `history`, `error`, `interrupt_ack`, `settings_updated`
- [ ] Implement JSON serialization/deserialization with validation
- [ ] Write unit tests for all message types

#### 1.3 WebSocket Server
- [ ] Implement WebSocket server in `internal/server/`:
  - Listen on `127.0.0.1:8085`
  - Accept WebSocket upgrade on `/ws`
  - Handle ping/pong heartbeat (30s interval)
  - Route incoming messages by `type` field to appropriate handlers
  - Support multiple concurrent connections (for future multi-window)
  - Broadcast outgoing messages to all connected clients
- [ ] Write integration test: connect, send message, receive response

#### 1.4 SQLite Persistence
- [ ] Implement storage layer in `internal/storage/`:
  - Database initialization with schema from Architecture doc Section 11.1
  - CRUD operations for: agents, messages, endpoints, settings
  - Paginated message history queries
  - WAL mode enabled for resilience
- [ ] Write unit tests for all CRUD operations

#### 1.5 Verification
- [ ] Run backend: `go run cmd/zeclaw/main.go`
- [ ] Connect with `wscat` or a test client
- [ ] Send `get_agents` → receive empty `agent_list`
- [ ] Create an agent via `create_agent` → verify in SQLite

---

## Phase 2: Flutter Frontend Shell

### Objective
Build the Flutter app skeleton: navigation, theme, WebSocket client, and placeholder screens.

### Steps

#### 2.1 Project Configuration
- [ ] Configure `pubspec.yaml` with dependencies:
  - `flutter_riverpod` - State management
  - `go_router` - Navigation/routing
  - `web_socket_channel` - WebSocket client
  - `google_fonts` - Inter & JetBrains Mono fonts
  - `flutter_markdown` - Markdown rendering
  - `flutter_syntax_view` or `highlight` - Code syntax highlighting

#### 2.2 Theme & Design System
- [ ] Implement dark theme in `lib/core/theme/`:
  - Color palette from UI/UX doc Section 1.1
  - Typography scale using Inter + JetBrains Mono
  - Component themes: AppBar, BottomNavigationBar, Card, Input, Button
  - Spacing constants

#### 2.3 Navigation Shell
- [ ] Implement bottom navigation bar with three tabs (Chat, Dashboard, Settings)
- [ ] Configure GoRouter with tab-based navigation
- [ ] Create placeholder screens for each tab
- [ ] Implement tab switching with crossfade animation

#### 2.4 WebSocket Client
- [ ] Implement WebSocket client in `lib/core/websocket/`:
  - Connect to `ws://localhost:8085/ws`
  - Serialize/deserialize message envelopes
  - Auto-reconnect with exponential backoff
  - Connection state tracking (connecting, connected, disconnected)
  - Message queue for buffering during disconnection
- [ ] Create Riverpod provider for WebSocket connection state
- [ ] Display connection status indicator in app bar

#### 2.5 Core Data Models
- [ ] Define Dart models mirroring the protocol in `lib/core/models/`:
  - `Agent`, `Message`, `Endpoint`, `Settings`
  - `Envelope`, `MessageType` enum
  - JSON serialization with `json_serializable` or manual `fromJson`/`toJson`

#### 2.6 Verification
- [ ] App launches on emulator with bottom nav and three placeholder screens
- [ ] WebSocket client connects to running Go backend
- [ ] Connection state shown in UI (connected/disconnected indicator)

---

## Phase 3: Agent Engine (Single Agent)

### Objective
Implement a single working agent: LLM integration, tool execution, and basic chat.

### Steps

#### 3.1 LLM Client (Go Backend)
- [ ] Implement LLM client interface in `internal/llm/`:

```go
type Client interface {
    ChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}
```

- [ ] Implement **Cloud API client** (OpenAI-compatible):
  - HTTP client with streaming SSE parsing
  - Bearer token authentication
  - Tool/function calling support
  - Error handling: rate limits, timeouts, invalid responses
- [ ] Implement **Ollama client**:
  - HTTP client with NDJSON streaming
  - `/api/chat` endpoint
  - Tool calling support (Ollama 0.4+)
- [ ] Write unit tests with mocked HTTP responses
- [ ] Write integration test against a real endpoint (optional, skippable in CI)

#### 3.2 Tool Executors (Go Backend)
- [ ] Implement tool interface in `internal/tools/`:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, input map[string]interface{}) (string, error)
}
```

- [ ] Implement `ShellTool`:
  - Execute commands via `os/exec.CommandContext`
  - Set process group (`Setpgid`) for clean termination
  - Capture combined stdout/stderr
  - Enforce 60s timeout
  - Truncate output to 10KB
- [ ] Implement `ReadFileTool`:
  - Read file contents via `os.ReadFile`
  - Restrict to allowed directories
  - Truncate to 50KB
- [ ] Implement `ToolRegistry`:
  - Register tools
  - Look up by name
  - Generate tool definitions for LLM (JSON schema format)
- [ ] Write unit tests for each tool

#### 3.3 Agent Runtime (Go Backend)
- [ ] Implement agent state machine in `internal/agent/`:
  - Agent struct with: ID, config, state, context (messages), LLM client, tools
  - State management: CREATED → IDLE → THINKING → EXECUTING → IDLE
  - Agent loop running in a goroutine (see Architecture doc Section 4.3)
  - Status broadcasting on every state transition
- [ ] Implement `AgentManager`:
  - Create agent (from `create_agent` message)
  - Route messages to correct agent
  - List agents
  - Delete agent (graceful shutdown)
- [ ] Wire everything together in WebSocket message handlers:
  - `user_message` → route to agent → agent processes → stream response back
  - `create_agent` → create agent → broadcast `agent_list`
- [ ] Write integration test: send message, receive LLM response with tool calls

#### 3.4 Basic Chat UI (Flutter)
- [ ] Implement Chat screen in `lib/features/chat/`:
  - Message list (scrollable, auto-scroll to bottom)
  - User message bubble (right-aligned, blue-tinted)
  - Agent message bubble (left-aligned, surface-colored)
  - Input bar with text field and send button
  - Send message via WebSocket
  - Receive and display agent responses
- [ ] Implement Riverpod providers:
  - `messagesProvider` - list of messages for active agent
  - `activeAgentProvider` - currently selected agent
- [ ] Simple text rendering (no Markdown yet, no code blocks yet)

#### 3.5 Verification
- [ ] Type a message → agent responds via LLM
- [ ] Agent can execute `shell` tool and return output
- [ ] Agent can use `read_file` tool and return file contents
- [ ] Full round-trip: Flutter → WebSocket → Go → LLM → Tool → Go → WebSocket → Flutter

---

## Phase 4: Real-Time Streaming & WebSocket Integration

### Objective
Implement real-time streaming of agent responses and granular status updates.

### Steps

#### 4.1 Streaming Agent Responses (Go Backend)
- [ ] Modify agent loop to stream LLM response chunks via WebSocket:
  - Each chunk sends `agent_message` with `is_final: false`
  - Final chunk sends `agent_message` with `is_final: true`
- [ ] Stream status updates at each state transition:
  - THINKING: `{ status: "working", detail: "Thinking..." }`
  - EXECUTING: `{ status: "working", detail: "Running shell command: ls -la" }`
  - IDLE: `{ status: "idle" }`
- [ ] Stream tool execution progress:
  - Start: `{ tool: "shell", input: "...", status: "running" }`
  - Complete: `{ tool: "shell", output: "...", status: "done" }`
  - Error: `{ tool: "shell", error: "...", status: "error" }`

#### 4.2 Streaming UI (Flutter)
- [ ] Implement streaming text rendering in message bubbles:
  - Text appears incrementally (word-by-word or character-by-character)
  - Typing indicator (three dots) shown before first chunk arrives
- [ ] Implement status chips within agent messages:
  - Inline chips showing tool execution status
  - Color-coded: blue (running), green (done), red (error)
- [ ] Implement agent status bar below top app bar:
  - Shows current agent name + status
  - Pulsing amber when working
  - Green dot when idle
- [ ] Implement real-time status feed:
  - Collapsible section within agent messages
  - Monospace font, terminal-log style
  - Auto-scroll to latest entry

#### 4.3 Markdown & Code Rendering (Flutter)
- [ ] Add Markdown rendering to agent message bubbles
- [ ] Add code block rendering with syntax highlighting
- [ ] Add copy-to-clipboard button on code blocks

#### 4.4 Verification
- [ ] Send a message → see typing indicator → text streams in word-by-word
- [ ] Agent uses a tool → status chip appears inline (blue → green)
- [ ] Status bar updates in real-time during agent work
- [ ] Status feed shows granular steps: "Starting task..." → "Running command..." → "Done"

---

## Phase 5: Interruptible Execution

### Objective
Implement the critical interrupt mechanism: users can send messages while agents are working.

### Steps

#### 5.1 Interrupt Infrastructure (Go Backend)
- [ ] Add interrupt channel to Agent struct:

```go
type Agent struct {
    // ...
    interruptChan chan *InterruptMessage  // buffered channel, size 1
    currentCancel context.CancelFunc      // cancels current tool execution
}
```

- [ ] Implement interrupt check points in agent loop:
  - Before each LLM API call
  - Before each tool execution
  - During tool execution (via context cancellation)
- [ ] Implement interrupt handler:
  - Cancel current context (`currentCancel()`)
  - Kill running shell processes (SIGTERM → SIGKILL after 5s)
  - Append interrupt metadata to agent context
  - Append new user message to agent context
  - Re-enter THINKING state
- [ ] Send `interrupt_ack` message to frontend

#### 5.2 WebSocket Interrupt Routing (Go Backend)
- [ ] Handle `interrupt` message type:
  - Extract `agent_id` and `content`
  - Push onto target agent's `interruptChan`
  - If channel is full (agent already has pending interrupt), replace
- [ ] Handle `stop_agent` message type:
  - Send interrupt with empty content (just cancel, don't re-process)

#### 5.3 Interrupt UI (Flutter)
- [ ] Modify input bar behavior when agent is working:
  - Show lightning bolt (⚡) icon on send button (or separate interrupt button)
  - Typing a message while agent works sends it as `interrupt` type
- [ ] Handle `interrupt_ack` in UI:
  - Previous agent message gets "Interrupted" status chip (grey)
  - User's interrupt message appears in chat
  - New agent response begins streaming
- [ ] Brief amber edge flash animation on interrupt acknowledgment

#### 5.4 Context Preservation
- [ ] Verify interrupted tool output is saved with "[INTERRUPTED]" suffix
- [ ] Verify system message is injected: "The user interrupted the previous task."
- [ ] Verify LLM receives full context including interruption history
- [ ] Test that agent correctly adapts behavior after interrupt

#### 5.5 Verification
- [ ] Start a long shell command (e.g., `sleep 30`)
- [ ] Send interrupt message while it's running
- [ ] Shell process is killed within 5 seconds
- [ ] Agent acknowledges interrupt and responds to new message
- [ ] Chat UI shows: original task → "Interrupted" chip → user interrupt → new agent response
- [ ] Interrupt during LLM streaming also works

---

## Phase 6: Multi-Agent System

### Objective
Support multiple agents with context switching and independent operation.

### Steps

#### 6.1 Multi-Agent Backend
- [ ] Ensure `AgentManager` supports multiple concurrent agents:
  - Thread-safe agent registry (`sync.RWMutex`)
  - Each agent runs in its own goroutine
  - Independent state machines, contexts, and interrupt channels
- [ ] Implement agent hierarchy:
  - `main` type: only one allowed, first agent created
  - `sub` type: multiple allowed, reference `parent_id`
- [ ] Implement `delete_agent`:
  - Cancel agent's goroutine
  - Clean up resources
  - Optionally cascade-delete subagents
  - Broadcast updated `agent_list`

#### 6.2 Agent Selector (Flutter)
- [ ] Implement agent selector dropdown in Chat screen top bar:
  - List all agents with name + status dot
  - Currently selected agent highlighted
  - Tap to switch active agent
- [ ] On agent switch:
  - Load selected agent's message history (from local cache or request via WS)
  - Update status bar to reflect selected agent's state
  - Update input bar (show interrupt if new agent is working)
  - Sam (previous agent) continues working in background
- [ ] Add "Create New Agent" option at bottom of dropdown

#### 6.3 Create Agent UI (Flutter)
- [ ] Implement "Create Agent" bottom sheet:
  - Fields: Name, System Prompt, Type (Main/Sub), Parent Agent (dropdown), Tools (checkboxes), LLM Endpoint (dropdown)
  - Validation: name required, system prompt required, unique name
  - Submit sends `create_agent` via WebSocket
- [ ] On success: auto-switch to new agent in chat

#### 6.4 Verification
- [ ] Create Main Agent → chat with it
- [ ] Create Subagent → switch to it → chat with it
- [ ] Switch back to Main Agent → see its conversation preserved
- [ ] Both agents can work simultaneously (start tasks on both, check Dashboard)
- [ ] Delete a subagent → it disappears from selector and dashboard

---

## Phase 7: Dashboard & Monitoring

### Objective
Build the Dashboard screen for monitoring all agents and their background tasks.

### Steps

#### 7.1 Dashboard Data (Go Backend)
- [ ] Implement periodic agent status broadcasting:
  - Every 2 seconds (or on state change), broadcast full agent status list
  - Include: agent ID, name, type, parent, state, current task description, duration
- [ ] Track task progress:
  - Each tool execution step logged with status (pending/running/done/error)
  - Task timeline stored in memory (not persisted, reconstructed on restart)

#### 7.2 Dashboard UI (Flutter)
- [ ] Implement Dashboard screen in `lib/features/dashboard/`:
  - **Active Agents** section: list of agent cards
  - **Background Tasks** section: step-by-step task progress logs
- [ ] Implement `AgentCard` widget:
  - Agent name, type, parent, status dot
  - Current status text (streaming from backend)
  - Duration timer (for active tasks)
  - Action buttons: [View Chat], [Stop], [Retry], [Dismiss] (contextual)
- [ ] Implement task timeline widget:
  - Steps with status icons: ✓ (done), 🔄 (running), ○ (pending), ✗ (error)
  - Auto-updates in real-time via WebSocket
- [ ] Wire action buttons:
  - [View Chat] → navigate to Chat with agent selected
  - [Stop] → send `stop_agent` via WebSocket
  - [Retry] → send `retry_agent` via WebSocket
  - [Dismiss] → clear error state locally

#### 7.3 Empty State
- [ ] When no agents are running: centered message + "Create Agent" CTA button

#### 7.4 Verification
- [ ] Dashboard shows all agents with correct statuses
- [ ] Start a task → Dashboard shows live progress
- [ ] Tap [View Chat] → navigates to Chat with that agent
- [ ] Tap [Stop] → agent's task is interrupted
- [ ] Agent errors → red card with [Retry] and [Dismiss] buttons

---

## Phase 8: Settings & LLM Configuration

### Objective
Build the Settings screen for managing LLM endpoints and app preferences.

### Steps

#### 8.1 Settings Backend (Go Backend)
- [ ] Implement settings CRUD via WebSocket messages:
  - `update_settings` → write to SQLite → broadcast `settings_updated`
- [ ] Implement endpoint management:
  - Create, update, delete LLM endpoints
  - Set default endpoint
  - API key encryption using a device-derived key (simplified for v1: use a static key, improve in v2 with Android Keystore)
- [ ] Implement endpoint connection test:
  - New message type: `test_endpoint` → attempt LLM call → return success/failure

#### 8.2 Settings UI (Flutter)
- [ ] Implement Settings screen in `lib/features/settings/`:
  - **LLM Configuration** section:
    - Default provider dropdown (Cloud API / Ollama)
    - Cloud API fields: endpoint URL, API key (masked + reveal toggle), model name
    - Ollama fields: host URL, model name, [Test Connection] button
  - **Saved Endpoints** section:
    - List of configured endpoints with [Default], [Edit], [Delete] actions
    - [+ Add Endpoint] button
  - **App Preferences** section:
    - Font size selector
    - Haptic feedback toggle
    - Status feed visibility toggle
  - **Backend** section:
    - Go binary status indicator
    - Port display
    - [Restart Backend] button
  - **About** section:
    - Version number
    - [View Logs], [Export Data], [Reset All] buttons
- [ ] Persist settings via WebSocket → Go backend → SQLite

#### 8.3 Onboarding Flow
- [ ] Implement first-launch onboarding (3 screens):
  1. Welcome to Zeclaw
  2. Configure your first LLM endpoint
  3. Name your Main Agent
- [ ] Store `onboarding_complete` flag in settings
- [ ] Skip onboarding on subsequent launches

#### 8.4 Verification
- [ ] Add a Cloud API endpoint → it appears in saved list
- [ ] Set it as default → agents use it for LLM calls
- [ ] Add an Ollama endpoint → test connection → success/failure displayed
- [ ] Change font size → UI updates
- [ ] First launch → onboarding → Chat screen
- [ ] Second launch → straight to Chat screen

---

## Phase 9: Polish, Testing & Hardening

### Objective
Refine the UI, add comprehensive tests, and harden the system for reliability.

### Steps

#### 9.1 UI Polish
- [ ] Implement all animations from UI/UX doc Section 6.1:
  - Tab switch crossfade
  - Bottom sheet slide
  - Message appear animation
  - Status chip scale-in
  - Agent status pulse
  - Interrupt edge flash
- [ ] Implement error states from UI/UX doc Section 6.2:
  - Backend offline banner
  - LLM unreachable toast
  - WebSocket disconnected banner with auto-retry indicator
- [ ] Implement empty states from UI/UX doc Section 6.3
- [ ] Accessibility pass:
  - Semantic labels on all interactive elements
  - 48x48dp minimum touch targets
  - System font scaling support
  - WCAG AA contrast verification

#### 9.2 Testing - Go Backend
- [ ] Unit tests for all packages:
  - `protocol/` - message serialization (already done in Phase 1)
  - `agent/` - state machine transitions, interrupt handling
  - `tools/` - shell execution, file reading (already done in Phase 3)
  - `llm/` - mock LLM client responses
  - `storage/` - CRUD operations (already done in Phase 1)
  - `server/` - WebSocket message routing
- [ ] Integration tests:
  - Full agent loop: message → LLM → tool → response
  - Interrupt during execution
  - Multiple concurrent agents
  - Reconnection after WebSocket drop
- [ ] Aim for >80% code coverage on critical paths (agent, tools, server)

#### 9.3 Testing - Flutter Frontend
- [ ] Widget tests:
  - Chat screen: message rendering, sending, streaming
  - Dashboard: agent cards, action buttons
  - Settings: form validation, endpoint CRUD
- [ ] Integration tests (Flutter integration_test):
  - Full user flow: launch → onboarding → chat → receive response
  - Agent switching
  - Interrupt flow
- [ ] Golden tests for key screens (optional, nice-to-have)

#### 9.4 Hardening
- [ ] WebSocket reconnection stress test:
  - Kill Go backend → Flutter reconnects → state re-synced
- [ ] Agent concurrency stress test:
  - 5 agents running simultaneously → no deadlocks or panics
- [ ] Memory leak check:
  - Long conversation (500+ messages) → no excessive memory growth
- [ ] Shell execution security audit:
  - Verify process group kills work reliably
  - Verify timeout enforcement
  - Test with long-running and zombie processes
- [ ] Input validation hardening:
  - Fuzz WebSocket message parsing
  - Reject malformed messages gracefully

#### 9.5 Performance
- [ ] Profile Go backend memory and CPU usage
- [ ] Optimize SQLite queries with proper indexing
- [ ] Implement message pagination (lazy loading in chat scroll)
- [ ] Limit in-memory message cache to last 200 messages per agent

---

## Phase 10: Packaging & Distribution

### Objective
Cross-compile the Go backend for Android and package the complete app.

### Steps

#### 10.1 Go Binary Cross-Compilation
- [ ] Set up cross-compilation for Android ARM targets:

```bash
# ARM64 (most modern Android devices)
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o zeclaw-backend-arm64 ./cmd/zeclaw/

# ARM32 (older devices, fallback)
GOOS=android GOARCH=arm CGO_ENABLED=0 go build -o zeclaw-backend-arm ./cmd/zeclaw/
```

- [ ] Verify binary runs on Android emulator
- [ ] Create build script (`scripts/build-backend.sh`) that produces both architectures
- [ ] Test with `modernc.org/sqlite` (pure Go, no CGO) — confirm it works with `CGO_ENABLED=0`

#### 10.2 Flutter-Go Integration
- [ ] Implement platform channel (MethodChannel) to:
  - Bundle Go binary as an asset in the Flutter APK
  - Extract binary to app's data directory on first launch
  - Start Go binary as a subprocess with correct permissions
  - Monitor subprocess health (restart on crash)
  - Stop subprocess on app terminate
- [ ] Alternative approach (if subprocess is restricted):
  - Use `gomobile` to compile Go as a shared library (`.so`)
  - Call via FFI/MethodChannel
  - (Evaluate in Phase 10; subprocess is preferred for simplicity)

#### 10.3 APK Packaging
- [ ] Configure `build.gradle` for:
  - Minimum SDK: API 26 (Android 8.0)
  - Target SDK: API 34
  - Multi-ABI support (arm64-v8a, armeabi-v7a)
- [ ] Bundle Go binaries for each ABI in `assets/`
- [ ] Implement first-launch binary extraction logic
- [ ] Build release APK: `flutter build apk --release`
- [ ] Build App Bundle: `flutter build appbundle --release`

#### 10.4 Testing on Real Devices
- [ ] Test on at least 2 physical devices:
  - Modern flagship (ARM64)
  - Mid-range device (check performance)
- [ ] Verify Go backend starts and serves WebSocket on real device
- [ ] Full end-to-end test: onboarding → configure LLM → chat → interrupt → multi-agent

#### 10.5 Distribution
- [ ] Generate signed APK for sideloading
- [ ] (Future) Google Play Store listing preparation

---

## Phase 11: Doctor & Approval Gates

> **Goal:** Add system diagnostics and human-in-the-loop approval for sensitive agent actions.
> **Duration:** 3-4 days
> **Depends on:** Phase 3 (Agent Engine), Phase 8 (Settings)
> **Architecture Reference:** ARCHITECTURE.md §13 (Doctor), §17 (Approval Gates)

### 11.1 Doctor / Diagnostics Engine

- [ ] Create `doctor/` package in Go backend
- [ ] Implement diagnostic check runner:
  - [ ] Configuration validation (settings present, endpoint URLs valid, API keys non-empty)
  - [ ] Provider health checks (HTTP connectivity test with 10s timeout per endpoint)
  - [ ] Ollama detection (check if running, list available models)
  - [ ] Tool health checks (verify `sh` available, workspace directory exists)
  - [ ] System health (Go backend alive, SQLite integrity via `PRAGMA integrity_check`, storage free, memory usage)
- [ ] Define `doctor_report` response schema (see ARCHITECTURE.md §13.2)
- [ ] Register `run_doctor` WebSocket message handler
- [ ] Emit `doctor_report` message back to frontend with `overall_status` (healthy/degraded/error) and `suggestions` array
- [ ] **Safety: Doctor is read-only** — never auto-modifies configuration
- [ ] Write unit tests for each diagnostic check category

### 11.2 Approval Gate System

- [ ] Create `approval/` package in Go backend
- [ ] Create `approval_requests` SQLite table (see ARCHITECTURE.md §20.2)
- [ ] Implement risk classifier:
  - [ ] Parse shell command inputs against `highRiskPatterns` regex list (rm -rf, sudo, kill, chmod, mkfs, dd, etc.)
  - [ ] Classify `read_file` outside workspace as HIGH
  - [ ] Low/Medium actions proceed immediately
  - [ ] High/Critical actions → pause agent, emit `approval_request`
- [ ] Add `WAITING_APPROVAL` sub-state to agent state machine
- [ ] Register `approve_action` WebSocket message handler
- [ ] On approval: resume tool execution
- [ ] On denial: inject "User denied this action" into agent context, agent adapts
- [ ] Implement 5-minute approval timeout → auto-deny with notification
- [ ] Write unit tests for risk classification and approval flow

### 11.3 Flutter UI for Doctor & Approvals

- [ ] Add "Run Doctor" button in Settings screen
- [ ] Create `DoctorReportScreen` — renders sections with pass/fail indicators, color-coded status, suggestions list
- [ ] Create `ApprovalDialog` widget — modal bottom sheet showing action details, risk level badge, Approve/Deny buttons
- [ ] Wire `approval_request` WebSocket message to show `ApprovalDialog` immediately (even if user is on another screen)
- [ ] Add pending approval badge to Dashboard for any unresolved approvals

#### Verification
- [ ] `run_doctor` returns correct report for healthy system
- [ ] `run_doctor` reports degraded when Ollama is not running
- [ ] High-risk shell command (`rm -rf /`) triggers approval request
- [ ] User can approve/deny from the dialog
- [ ] Denied action causes agent to adapt its plan
- [ ] Approval timeout auto-denies after 5 minutes

---

## Phase 12: Heartbeat, Cron & Memory

> **Goal:** Add scheduled awareness (heartbeat), recurring tasks (cron), and persistent agent memory.
> **Duration:** 5-7 days
> **Depends on:** Phase 3 (Agent Engine), Phase 5 (Interrupts)
> **Architecture Reference:** ARCHITECTURE.md §9 (Heartbeat), §10 (Cron), §11 (Memory)

### 12.1 Heartbeat System

- [ ] Create `heartbeat/` package in Go backend
- [ ] Create `heartbeat_config` SQLite table (see ARCHITECTURE.md §20.2)
- [ ] Implement per-agent heartbeat scheduler:
  - [ ] `time.NewTicker` per agent with configurable interval (default 30m)
  - [ ] Active hours filter (skip heartbeat outside configured window)
  - [ ] Only fire when agent is IDLE (skip if EXECUTING/THINKING)
  - [ ] Support `target: "chat"` (surface to user) vs `"silent"` (log only)
- [ ] Implement heartbeat decision pipeline:
  - [ ] Layer 1: Cheap deterministic checks (background task status, file changes, error queue)
  - [ ] Layer 2: Rule evaluation (threshold breaches)
  - [ ] Layer 3: Escalation gate — only call LLM if issues found
  - [ ] Layer 4: Action dispatch (surface to chat, auto-remediate, log)
- [ ] Support per-agent `HEARTBEAT.md` checklist file in workspace
- [ ] Register `update_heartbeat` WebSocket message handler
- [ ] Emit `heartbeat_result` message to frontend
- [ ] Heartbeat respects Android battery optimization — reduce frequency when battery < 20%
- [ ] Write unit tests for scheduler timing, decision pipeline, and active hours filter

### 12.2 Cron / Scheduled Tasks

- [ ] Create `cron/` package in Go backend
- [ ] Create `cron_tasks` SQLite table (see ARCHITECTURE.md §20.2)
- [ ] Implement cron expression parser (standard 5-field: minute hour day month weekday)
- [ ] Implement interval syntax parser ("every 15m", "every 2h", "every 1d")
- [ ] Implement one-shot cron (run once at `run_at` timestamp, then auto-delete)
- [ ] Implement execution modes:
  - [ ] `main`: inject task as `user_message` into agent's existing session
  - [ ] `isolated`: spin up fresh context with only system prompt + task
- [ ] Register WebSocket handlers: `create_cron`, `update_cron`, `delete_cron`, `get_crons`
- [ ] Emit `cron_triggered` notification to frontend when cron fires
- [ ] Persist `last_run` timestamp after each execution
- [ ] Write unit tests for cron expression parsing, interval parsing, one-shot behavior

### 12.3 Agent Memory System

- [ ] Create `memory/` package in Go backend
- [ ] Create `memories` SQLite table (see ARCHITECTURE.md §20.2)
- [ ] Implement memory tools available to LLM:
  - [ ] `memory_store` — save a memory with content, category, importance
  - [ ] `memory_search` — keyword search through memories (SQLite FTS5 or LIKE)
  - [ ] `memory_forget` — delete a specific memory by ID
  - [ ] `memory_summarize` — condense memories older than N days into summaries
  - [ ] `memory_list` — list recent memories filtered by category
- [ ] Register memory tools in Tool Registry alongside shell/read_file
- [ ] Implement memory injection into LLM context:
  - [ ] Before each LLM call, search memories relevant to current conversation
  - [ ] Inject top 5 memories (by importance) into system prompt
- [ ] Implement memory decay:
  - [ ] Memories not accessed for 30 days → reduce importance by 10%/week
  - [ ] Memories with importance < 0.1 → candidate for auto-summarization
  - [ ] Run decay check as part of heartbeat cycle (piggyback, no separate scheduler)
- [ ] Register `memory_search` WebSocket handler for frontend memory browser
- [ ] Write unit tests for store/search/forget/summarize/decay

### 12.4 Flutter UI for Heartbeat, Cron & Memory

- [ ] Add Heartbeat configuration section in agent edit screen:
  - [ ] Toggle enabled/disabled
  - [ ] Interval picker (15m, 30m, 1h, 2h, custom)
  - [ ] Active hours range picker
  - [ ] Target selector (chat vs silent)
- [ ] Add Cron management UI:
  - [ ] "Scheduled Tasks" tab in agent detail view
  - [ ] Create cron dialog: expression input, task description, run mode toggle
  - [ ] Cron list with enable/disable toggle and delete
  - [ ] Visual next-run indicator
- [ ] Add Memory browser UI:
  - [ ] Accessible from agent detail view → "Memory" tab
  - [ ] Search bar for keyword search
  - [ ] Memory list showing content, category badge, importance bar
  - [ ] Swipe-to-delete on individual memories
  - [ ] Memory statistics (total count, by category)

#### Verification
- [ ] Heartbeat fires at configured interval when agent is idle
- [ ] Heartbeat skips when outside active hours
- [ ] Heartbeat surfaces alert to chat when issue detected
- [ ] Cron task fires at correct time (test with "every 1m")
- [ ] One-shot cron fires once then disappears
- [ ] Isolated cron task doesn't share context with main conversation
- [ ] Agent can store and recall memories across sessions
- [ ] Memory search returns relevant results
- [ ] Memory decay reduces importance over time

---

## Phase 13: Skills, Hooks & Workspace

> **Goal:** Add declarative skill definitions, event-driven hooks, and per-agent workspace management.
> **Duration:** 4-5 days
> **Depends on:** Phase 3 (Agent Engine), Phase 12 (Memory)
> **Architecture Reference:** ARCHITECTURE.md §12 (Skills), §14 (Hooks), §15 (Workspace)

### 13.1 Skills System

- [ ] Create `skills/` package in Go backend
- [ ] Create `skills` SQLite table (see ARCHITECTURE.md §20.2)
- [ ] Implement skill file parser:
  - [ ] Parse YAML frontmatter (name, description, tools_required)
  - [ ] Extract markdown body as skill content
- [ ] Implement skill loading pipeline:
  - [ ] Load built-in skills from app bundle (default: general-assistant)
  - [ ] Load workspace skills from agent's `workspace/skills/` directory
  - [ ] Filter by agent's tool access (skip skills requiring unavailable tools)
- [ ] Inject skills into system prompt (after system prompt, before memories)
- [ ] Implement skill precedence: agent system prompt > workspace skills > built-in skills
- [ ] Register WebSocket handlers: `update_skill`, `get_skills`
- [ ] Write unit tests for skill parsing, loading, precedence

### 13.2 Hooks & Event-Driven Triggers

- [ ] Create `hooks/` package in Go backend
- [ ] Create `hooks` SQLite table (see ARCHITECTURE.md §20.2)
- [ ] Implement Hook Dispatcher:
  - [ ] Register hooks per event type per agent
  - [ ] When event occurs, find matching enabled hooks
  - [ ] Execute hook action:
    - [ ] `"message"` → inject message into agent's incoming queue
    - [ ] `"retry"` → re-run last failed action (with configurable max retries and backoff)
    - [ ] `"notify"` → send notification to frontend
    - [ ] `"agent_task"` → send a new task to the agent
  - [ ] Log hook execution to audit trail
- [ ] Emit `hook_fired` WebSocket message to frontend
- [ ] Integrate hook dispatcher with agent lifecycle events:
  - [ ] `on_task_complete` — fires when agent transitions to IDLE after EXECUTING
  - [ ] `on_task_error` — fires when agent transitions to ERROR
  - [ ] `on_file_change` — fires when workspace file is modified (fsnotify or periodic check)
  - [ ] `on_cron_fire` — fires when cron task triggers
  - [ ] `on_agent_idle` — fires when agent transitions to IDLE
  - [ ] `on_heartbeat` — fires on heartbeat tick
- [ ] Write unit tests for hook registration, event matching, action execution

### 13.3 Agent Workspace Management

- [ ] Create `workspace/` package in Go backend
- [ ] Implement workspace initialization:
  - [ ] On agent creation, create workspace directory at `/data/data/com.zeclaw.app/files/workspaces/{agent-uuid}/`
  - [ ] Create subdirectories: `skills/`, `outputs/`, `temp/`
  - [ ] Create default files: `MEMORY.md`, `SESSION-STATE.md`
- [ ] Create shared workspace at `workspaces/shared/` with `AGENTS.md` and `knowledge/`
- [ ] Implement workspace access rules:
  - [ ] Agent can only read/write within its own workspace
  - [ ] All agents can read `shared/` directory
  - [ ] Shell tool working directory defaults to agent's workspace
- [ ] Implement workspace cleanup:
  - [ ] On agent deletion, archive workspace (don't delete immediately)
  - [ ] `temp/` directory cleared on agent restart
- [ ] Write unit tests for workspace creation, access rules, cleanup

### 13.4 Flutter UI for Skills, Hooks & Workspace

- [ ] Add Skills management UI:
  - [ ] "Skills" tab in agent detail view
  - [ ] List loaded skills with name, description, enabled toggle
  - [ ] Create/edit skill dialog with markdown editor
  - [ ] Skill detail view showing full content
- [ ] Add Hooks management UI:
  - [ ] "Hooks" tab in agent detail view
  - [ ] Create hook: event dropdown, action type dropdown, config fields
  - [ ] Hook list with enable/disable toggle
  - [ ] Hook activity log (last 10 firings)
- [ ] Add Workspace browser UI:
  - [ ] "Files" tab in agent detail view
  - [ ] Simple file tree view of workspace directory
  - [ ] Tap file to view contents (read-only for now)
  - [ ] File metadata (size, modified date)

#### Verification
- [ ] Skill loaded from workspace appears in agent's system prompt
- [ ] Skills requiring unavailable tools are skipped
- [ ] `on_task_complete` hook fires when agent finishes a task
- [ ] `on_task_error` hook with retry action retries the failed operation
- [ ] New agent gets workspace with correct directory structure
- [ ] Agent shell commands run in workspace directory by default
- [ ] Workspace browser shows correct file tree

---

## Phase 14: Audit Trail & Agent Behavior

> **Goal:** Add comprehensive action logging and AGENTS.md-style behavior definitions.
> **Duration:** 3-4 days
> **Depends on:** Phase 11 (Approval Gates), Phase 13 (Hooks)
> **Architecture Reference:** ARCHITECTURE.md §16 (Audit Trail), §22 (Comparison Matrix)

### 14.1 Audit Trail System

- [ ] Create `audit/` package in Go backend
- [ ] Create `audit_log` SQLite table (see ARCHITECTURE.md §20.2)
- [ ] Implement audit logger — append-only writes for all significant events:
  - [ ] `tool_call` — tool name, input, output (truncated to 500 chars), duration, success/failure
  - [ ] `llm_request` — model, token count (input/output), latency, cost estimate
  - [ ] `state_change` — agent ID, from_state, to_state, trigger
  - [ ] `interrupt` — agent ID, interrupted task, new message
  - [ ] `memory_write` — memory ID, category, content preview
  - [ ] `cron_fire` — cron ID, task, result
  - [ ] `heartbeat` — agent ID, checks summary, escalation decision
  - [ ] `approval_request` — action details, user decision, response latency
  - [ ] `hook_fire` — hook ID, event, action taken
  - [ ] `error` — category, message, stack trace (if applicable)
- [ ] Integrate audit logging into:
  - [ ] Tool executor (log before and after tool execution)
  - [ ] LLM client (log each request/response)
  - [ ] Agent runtime (log state transitions)
  - [ ] Approval manager (log requests and decisions)
  - [ ] Hook dispatcher (log each hook firing)
- [ ] Implement audit retention: auto-prune entries older than 30 days (configurable)
- [ ] Implement audit export: JSON export for entries before pruning
- [ ] Register `get_audit_log` WebSocket handler for frontend
- [ ] Write unit tests for audit logging, retention, and export

### 14.2 AGENTS.md — Agent Behavior Definitions

- [ ] Implement `AGENTS.md` support:
  - [ ] Global `AGENTS.md` file in shared workspace defines cross-agent coordination rules
  - [ ] Per-agent behavior rules injected into system prompt
  - [ ] Format:
    ```markdown
    # Agent Behavior Rules

    ## Proactive Initiative
    - If you notice a potential issue during a task, flag it immediately
    - If a task is blocked, suggest alternative approaches

    ## Guardrails
    - Never delete files without explicit user approval
    - Always explain what you're about to do before running shell commands
    - If unsure, ask the user instead of guessing

    ## Communication Style
    - Be concise but thorough
    - Use bullet points for multi-item responses
    ```
- [ ] Load `AGENTS.md` from shared workspace at agent initialization
- [ ] Inject into system prompt before skills and memories
- [ ] Allow editing via workspace browser UI

### 14.3 Flutter UI for Audit & Behavior

- [ ] Add Audit Log viewer:
  - [ ] Accessible from Settings > "Audit Log" or per-agent from Dashboard
  - [ ] Filterable by event type, agent, date range
  - [ ] Each entry expandable to show full data
  - [ ] Export button (JSON download)
  - [ ] Color-coded by event type (green=success, red=error, yellow=approval)
- [ ] Add AGENTS.md editor:
  - [ ] Accessible from Settings > "Agent Behavior Rules"
  - [ ] Full markdown editor
  - [ ] Preview pane showing how rules will appear in agent context

#### Verification
- [ ] Every tool call produces an audit log entry
- [ ] Every LLM request produces an audit log entry
- [ ] State transitions are logged
- [ ] Audit log entries older than 30 days are auto-pruned
- [ ] Export produces valid JSON file
- [ ] AGENTS.md content appears in agent system prompt
- [ ] Editing AGENTS.md takes effect on next agent interaction

---

## Dependency Map

This shows which phases depend on which:

```
Phase 0 (Setup)
  │
  ├──► Phase 1 (Go Backend Core)
  │      │
  │      ├──► Phase 3 (Agent Engine) ──► Phase 5 (Interrupts) ──► Phase 6 (Multi-Agent)
  │      │                                                              │
  │      └──► Phase 4 (Streaming) ◄────────────────────────────────────┘
  │                    │
  ├──► Phase 2 (Flutter Shell)                                          │
  │      │                                                              │
  │      ├──► Phase 3 (Agent Engine)                                    │
  │      │                                                              │
  │      ├──► Phase 7 (Dashboard) ◄─────────────────────────────────────┘
  │      │
  │      └──► Phase 8 (Settings)
  │
  └──► Phase 9 (Polish & Testing) ──► Phase 10 (Packaging)

  ── Agentic Feature Phases (post-MVP) ──────────────────────────

  Phase 3 + Phase 8 ──► Phase 11 (Doctor & Approval Gates)
                              │
  Phase 3 + Phase 5 ──► Phase 12 (Heartbeat, Cron & Memory)
                              │
  Phase 3 + Phase 12 ──► Phase 13 (Skills, Hooks & Workspace)
                              │
  Phase 11 + Phase 13 ──► Phase 14 (Audit Trail & Agent Behavior)
```

**Core critical path:** Phase 0 → 1 → 3 → 5 → 6 → 9 → 10
**Agentic feature path:** Phase 3 → 11 → 14 (Doctor/Approval → Audit) and Phase 5 → 12 → 13 → 14 (Heartbeat/Cron → Skills/Hooks → Audit)

> **Note:** Phases 11-14 are post-MVP agentic enhancements. The core app (Phases 0-10) should be fully functional before starting these. However, Phases 11-14 can be developed in parallel with Phase 9/10 polish work if resources allow.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **Go subprocess restricted on Android** | Medium | High | Fallback to `gomobile` shared library approach (Phase 10.2). Research Android app_process restrictions early. |
| **CGO required by some Go library** | Low | High | Use `modernc.org/sqlite` (pure Go). Avoid all CGO dependencies. Validate with `CGO_ENABLED=0` builds. |
| **Ollama tool calling limitations** | Medium | Medium | Implement a fallback "manual" tool calling parser for models that don't support native tool use. Parse JSON from model output. |
| **WebSocket stability on mobile** | Medium | Medium | Robust reconnection with exponential backoff. Foreground service for background operation. Message buffering. |
| **LLM response format inconsistency** | Medium | Low | Implement response sanitization layer. Handle partial JSON, missing fields, unexpected formats gracefully. |
| **Memory pressure on low-end devices** | Medium | Medium | Aggressive message pagination. Limit in-memory context window. Profile early on mid-range hardware. |
| **Shell command security** | Low | High | Process group isolation. Timeout enforcement. No root access. Restrict to app sandbox. Audit in Phase 9. |
| **Streaming UI performance (many updates)** | Low | Medium | Throttle status updates to max 10/second. Batch UI updates. Use `RepaintBoundary` in Flutter. |
| **Heartbeat battery drain** | Medium | Medium | Default to disabled. Enforce minimum 15m interval. Skip heartbeats when battery < 20%. Respect Android Doze mode. |
| **Cron drift on sleeping device** | Medium | Low | Use `AlarmManager` for exact timing. Accept that Android may defer cron tasks in Doze mode. Document limitation. |
| **Memory table growth unbounded** | Medium | Medium | Implement memory decay (Phase 12). Auto-summarize old memories. Cap at 1000 memories per agent with LRU eviction. |
| **Approval gate blocks agent indefinitely** | Low | Medium | 5-minute auto-deny timeout. Notification to user. Agent adapts to denial gracefully. |
| **Risk classifier false positives** | Medium | Low | Allow user to whitelist specific commands. Start conservative (more false positives). Refine patterns over time. |
| **Audit log storage growth** | Medium | Medium | 30-day auto-prune. Export before prune. Cap at 50MB per agent. Configurable retention period. |
| **Skill injection exceeds context window** | Low | High | Count tokens before injection. Prioritize by relevance. Warn user if total prompt exceeds 80% of model context window. |
