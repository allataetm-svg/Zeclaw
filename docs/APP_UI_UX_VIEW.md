# Zeclaw - App UI/UX View

> Dark mode, minimalist, Notion-like aesthetic. No clutter.

---

## Table of Contents

1. [Design System](#1-design-system)
2. [Navigation Structure](#2-navigation-structure)
3. [Screen Breakdown](#3-screen-breakdown)
4. [Component Library](#4-component-library)
5. [User Flows](#5-user-flows)
6. [Interaction Patterns](#6-interaction-patterns)

---

## 1. Design System

### 1.1 Color Palette

| Token               | Hex       | Usage                                      |
|----------------------|-----------|---------------------------------------------|
| `background-primary` | `#191919` | Main app background                         |
| `background-surface` | `#1E1E1E` | Cards, panels, bottom sheet surfaces        |
| `background-elevated`| `#252525` | Elevated cards, modals, dropdown menus      |
| `border-subtle`      | `#2E2E2E` | Dividers, input borders                     |
| `border-active`      | `#4A4A4A` | Focused input borders                       |
| `text-primary`       | `#EBEBEB` | Headings, body text                         |
| `text-secondary`     | `#999999` | Placeholder text, timestamps, labels        |
| `text-muted`         | `#666666` | Disabled text, subtle hints                 |
| `accent-primary`     | `#3B82F6` | Primary buttons, active tab indicator, links |
| `accent-success`     | `#22C55E` | Success states, "online" indicators          |
| `accent-warning`     | `#F59E0B` | Warning banners, "working" status            |
| `accent-error`       | `#EF4444` | Error states, destructive actions            |

### 1.2 Typography

- **Font Family:** `Inter` (primary), `JetBrains Mono` (code/terminal output)
- **Scale:**
  - H1: 24sp / Bold / `text-primary`
  - H2: 20sp / SemiBold / `text-primary`
  - H3: 16sp / SemiBold / `text-primary`
  - Body: 14sp / Regular / `text-primary`
  - Caption: 12sp / Regular / `text-secondary`
  - Code: 13sp / Regular / `JetBrains Mono` / `text-primary`

### 1.3 Spacing & Layout

- Base unit: **4dp**
- Standard padding: 16dp
- Card border radius: 12dp
- Button border radius: 8dp
- Input field height: 48dp
- Bottom nav height: 64dp

### 1.4 Iconography

- Style: Outlined, 24x24dp, 1.5dp stroke
- Library: Material Symbols Outlined (or Lucide Icons)
- Active state: `accent-primary`
- Inactive state: `text-muted`

---

## 2. Navigation Structure

### 2.1 Bottom Navigation Bar

A persistent bottom nav with three destinations. Follows Material Design 3 guidelines with a dark theme override.

```
+----------------------------------------------------+
|                                                    |
|                   [Screen Content]                 |
|                                                    |
+----------------------------------------------------+
|  [ Chat ]       [ Dashboard ]       [ Settings ]  |
|   (icon)           (icon)             (icon)       |
+----------------------------------------------------+
```

| Tab         | Icon              | Label       | Description                              |
|-------------|-------------------|-------------|------------------------------------------|
| Chat        | `chat_bubble`     | Chat        | Main agent conversation interface         |
| Dashboard   | `dashboard`       | Dashboard   | Monitor active agents and background tasks|
| Settings    | `settings`        | Settings    | LLM config, Ollama endpoints, preferences |

- **Active indicator:** Pill-shaped highlight behind active icon using `accent-primary` at 15% opacity, icon and label in `accent-primary`.
- **Inactive:** Icon and label in `text-muted`.
- **Haptic feedback:** Light haptic on tab switch.

---

## 3. Screen Breakdown

### 3.1 Chat Screen (Primary)

The main interface. Users talk to agents here.

#### Layout

```
+----------------------------------------------------+
|  [Agent Selector Dropdown ▼]        [+ New Agent]  |  <- Top App Bar
|----------------------------------------------------|
|                                                    |
|  Agent: Sam (Online)                               |  <- Agent Status Bar
|  Status: "Downloading dependency..."               |
|                                                    |
|----------------------------------------------------|
|                                                    |
|  [User Message Bubble - right aligned]             |
|                                                    |
|  [Agent Message Bubble - left aligned]             |
|    ├─ Streaming text...                            |
|    └─ [Status Chip: "Executing shell..."]          |
|                                                    |
|  [Agent Message Bubble - left aligned]             |
|    ├─ Code block with syntax highlighting          |
|    └─ [Status Chip: "Task complete ✓"]             |
|                                                    |
|  [Real-time Status Feed]                           |
|    "Starting task..."                              |
|    "Reading config file..."                        |
|    "Environment error encountered. Fixing..."      |
|    "Task complete."                                |
|                                                    |
+----------------------------------------------------+
|  [Interrupt ⚡]  [ Text Input ...        ] [Send ▶] |  <- Input Bar
+----------------------------------------------------+
|  [ Chat ]       [ Dashboard ]       [ Settings ]   |
+----------------------------------------------------+
```

#### Components

| Component             | Description                                                                                 |
|-----------------------|---------------------------------------------------------------------------------------------|
| **Agent Selector**    | Dropdown at top-left. Lists all created agents (Main Agent + Subagents). Shows name + status icon (green = idle, amber = working, red = error). Tapping switches the active conversation context. |
| **New Agent Button**  | `+` icon at top-right. Opens the "Create Agent" bottom sheet.                                |
| **Agent Status Bar**  | Slim bar below the top bar. Shows the currently selected agent's name, online status, and the latest real-time status message streamed from the backend. Animates with a subtle pulse when agent is actively working. |
| **Message List**      | Scrollable message area. User messages right-aligned (subtle blue-tinted bubble). Agent messages left-aligned (surface-colored bubble). Supports: plain text, Markdown rendering, code blocks with syntax highlighting and copy button, file references. |
| **Status Chips**      | Inline chips within agent messages showing granular execution steps. Color-coded: blue = in-progress, green = success, red = error, amber = warning. |
| **Real-Time Feed**    | Collapsible section within agent messages showing the live stream of status updates. Styled like a minimal terminal log with monospace font. Auto-scrolls to latest entry. |
| **Input Bar**         | Fixed at bottom above nav. Text field with send button. **Interrupt button** (lightning icon) appears when the agent is actively working -- pressing it sends the typed message as an interrupt, forcing the agent to pause and respond. |

#### Agent Selector Dropdown Detail

```
+----------------------------------+
|  🟢 Sam (Main Agent)            |  <- Currently selected (highlighted)
|  🟡 Bob (Working...)            |
|  🟢 Alice (Idle)                |
|  🔴 Debug-Bot (Error)           |
|----------------------------------|
|  [+ Create New Agent]            |
+----------------------------------+
```

### 3.2 Create Agent Bottom Sheet

Triggered from the `+` button on the Chat screen or the Dashboard.

```
+----------------------------------------------------+
|                  Create New Agent                   |
|----------------------------------------------------|
|                                                    |
|  Agent Name                                        |
|  [ __________________________________ ]            |
|                                                    |
|  System Prompt                                     |
|  [ __________________________________ ]            |
|  [ __________________________________ ]            |
|  [ __________________________________ ]            |
|                                                    |
|  Agent Type                                        |
|  ( ) Main Agent   (o) Subagent                     |
|                                                    |
|  Parent Agent (if Subagent)                        |
|  [ Select parent agent...  ▼ ]                     |
|                                                    |
|  Tools                                             |
|  [✓] Shell Execution                               |
|  [✓] Read File                                     |
|                                                    |
|  LLM Endpoint                                      |
|  [ Use default (from Settings) ▼ ]                 |
|                                                    |
|            [ Cancel ]    [ Create Agent ]           |
+----------------------------------------------------+
```

| Field              | Type        | Notes                                              |
|--------------------|-------------|-----------------------------------------------------|
| Agent Name         | Text input  | Required. Alphanumeric + hyphens. Max 32 chars.     |
| System Prompt      | Multiline   | Required. The persona/instructions for the agent.    |
| Agent Type         | Radio       | Main Agent or Subagent. Only one Main Agent allowed. |
| Parent Agent       | Dropdown    | Visible only when "Subagent" is selected.            |
| Tools              | Checkboxes  | Select which tools the agent can use.                |
| LLM Endpoint       | Dropdown    | Override per-agent or use global default.            |

### 3.3 Dashboard Screen

Overview of all agents and their current activity.

```
+----------------------------------------------------+
|  Dashboard                            [+ New Agent] |
|----------------------------------------------------|
|                                                    |
|  ACTIVE AGENTS (3)                                 |
|                                                    |
|  +------------------------------------------------+|
|  | 🟡 Sam (Main Agent)                            ||
|  | Status: "Installing npm packages..."           ||
|  | Task: "Set up React project"                   ||
|  | Duration: 2m 34s                               ||
|  | [View Chat] [Stop]                              ||
|  +------------------------------------------------+|
|                                                    |
|  +------------------------------------------------+|
|  | 🟢 Bob (Subagent of Sam)                       ||
|  | Status: Idle                                   ||
|  | Last task: "Linted codebase" (3m ago)          ||
|  | [View Chat]                                     ||
|  +------------------------------------------------+|
|                                                    |
|  +------------------------------------------------+|
|  | 🔴 Alice (Subagent of Sam)                     ||
|  | Status: Error - "Permission denied"            ||
|  | Task: "Read /etc/shadow"                       ||
|  | [View Chat] [Retry] [Dismiss]                   ||
|  +------------------------------------------------+|
|                                                    |
|  BACKGROUND TASKS                                  |
|                                                    |
|  +------------------------------------------------+|
|  | Sam > "Set up React project"                   ||
|  |  ├─ ✓ Created directory structure              ||
|  |  ├─ ✓ Initialized package.json                 ||
|  |  ├─ 🔄 Installing npm packages...             ||
|  |  └─ ○ Configure ESLint                         ||
|  +------------------------------------------------+|
|                                                    |
+----------------------------------------------------+
|  [ Chat ]       [ Dashboard ]       [ Settings ]   |
+----------------------------------------------------+
```

#### Components

| Component          | Description                                                                             |
|--------------------|-----------------------------------------------------------------------------------------|
| **Agent Cards**    | Each card shows: agent name, type, parent (if subagent), current status, current/last task, duration. Status indicator (colored dot). Action buttons contextual to state. |
| **View Chat**      | Navigates to Chat screen with that agent selected.                                       |
| **Stop**           | Sends an interrupt signal to halt the agent's current task.                              |
| **Retry**          | Re-executes the last failed task.                                                        |
| **Dismiss**        | Clears the error state.                                                                  |
| **Background Tasks** | A timeline/log view showing the step-by-step progress of each active task. Steps are color-coded: ✓ green (done), 🔄 amber (in-progress), ○ grey (pending), ✗ red (failed). |

### 3.4 Settings Screen

```
+----------------------------------------------------+
|  Settings                                          |
|----------------------------------------------------|
|                                                    |
|  LLM CONFIGURATION                                |
|                                                    |
|  Default Provider                                  |
|  [ Cloud API  ▼ ]                                  |
|                                                    |
|  ── Cloud API Settings ──                          |
|  API Endpoint URL                                  |
|  [ https://api.openai.com/v1 _________ ]          |
|  API Key                                           |
|  [ ●●●●●●●●●●●●●●●●  ] [👁]                      |
|  Model                                             |
|  [ gpt-4 __________________________ ]              |
|                                                    |
|  ── Ollama Settings ──                             |
|  Ollama Host                                       |
|  [ http://localhost:11434 __________ ]             |
|  Model                                             |
|  [ llama3 _________________________ ]              |
|  [Test Connection]                                 |
|                                                    |
|  SAVED ENDPOINTS                                   |
|  +------------------------------------------------+|
|  | "GPT-4 Cloud"    api.openai.com    [Default]   ||
|  | "Local Llama"    localhost:11434    [Edit] [Del]||
|  +------------------------------------------------+|
|  [ + Add Endpoint ]                                |
|                                                    |
|  ──────────────────────────────                    |
|                                                    |
|  APP PREFERENCES                                   |
|  Theme                        [ Dark (Only) ]      |
|  Font Size                    [ Medium ▼ ]          |
|  Haptic Feedback              [ ● On ]              |
|  Show Status Feed in Chat     [ ● On ]              |
|                                                    |
|  ──────────────────────────────                    |
|                                                    |
|  BACKEND                                           |
|  Go Binary Status             🟢 Running           |
|  Port                         8085                  |
|  [Restart Backend]                                 |
|                                                    |
|  ──────────────────────────────                    |
|                                                    |
|  ABOUT                                             |
|  Version                      0.1.0                 |
|  [View Logs]   [Export Data]   [Reset All]         |
|                                                    |
+----------------------------------------------------+
|  [ Chat ]       [ Dashboard ]       [ Settings ]   |
+----------------------------------------------------+
```

#### Settings Sections

| Section            | Fields & Controls                                                                  |
|--------------------|------------------------------------------------------------------------------------|
| **LLM Config**     | Provider dropdown (Cloud API / Ollama). Per-provider fields: endpoint URL, API key (masked with reveal toggle), model name. "Test Connection" button for Ollama. |
| **Saved Endpoints**| List of configured LLM endpoints. Each can be set as default, edited, or deleted. Agents can override with a specific endpoint. |
| **App Preferences**| Theme (locked to Dark for v1), font size selector, haptic feedback toggle, status feed visibility toggle. |
| **Backend**        | Shows Go binary process status (Running/Stopped/Error), configured port. "Restart Backend" button. |
| **About**          | App version, links to view raw logs, export conversation data, factory reset.       |

---

## 4. Component Library

### 4.1 Reusable Components

| Component            | Props / Variants                                                   | Usage                         |
|----------------------|--------------------------------------------------------------------|-------------------------------|
| `AgentAvatar`        | `name`, `status` (idle/working/error), `size` (sm/md/lg)          | Agent selector, cards, chat   |
| `StatusChip`         | `label`, `state` (info/success/warning/error/loading)              | Chat messages, dashboard      |
| `MessageBubble`      | `sender` (user/agent), `content`, `timestamp`, `statusChips[]`     | Chat message list             |
| `CodeBlock`          | `language`, `code`, `copyable`                                     | Inside message bubbles        |
| `StatusFeed`         | `entries[]`, `collapsed`, `autoScroll`                             | Chat, dashboard               |
| `AgentCard`          | `agent`, `actions[]`                                               | Dashboard                     |
| `InputBar`           | `onSend`, `onInterrupt`, `isAgentWorking`                          | Chat screen                   |
| `BottomSheet`        | `title`, `children`, `onDismiss`                                   | Create agent, confirmations   |
| `SectionHeader`      | `title`, `count?`, `action?`                                       | Dashboard, settings           |
| `SettingsRow`        | `label`, `control` (toggle/dropdown/text/button)                   | Settings screen               |
| `EndpointCard`       | `name`, `url`, `isDefault`, `onEdit`, `onDelete`                   | Settings saved endpoints      |

### 4.2 State Indicators

| State    | Visual                                                          |
|----------|-----------------------------------------------------------------|
| Idle     | Green dot (`accent-success`), static                            |
| Working  | Amber dot (`accent-warning`), pulsing animation                 |
| Error    | Red dot (`accent-error`), static                                |
| Offline  | Grey dot (`text-muted`), static                                 |
| Streaming| Animated typing indicator (three dots bouncing)                 |

---

## 5. User Flows

### 5.1 First Launch

```
App Start
  │
  ├─ Go backend binary starts automatically
  │
  ├─ Splash screen (1.5s, Zeclaw logo, dark bg)
  │
  ├─ Onboarding (first launch only):
  │    ├─ Screen 1: "Welcome to Zeclaw" - brief description
  │    ├─ Screen 2: "Configure your LLM" - set up first endpoint
  │    └─ Screen 3: "Meet your Main Agent" - create/name main agent
  │
  └─ Land on Chat screen with Main Agent selected
```

### 5.2 Send a Message (Agent Idle)

```
User types message in Input Bar
  │
  ├─ Taps Send (or presses Enter)
  │
  ├─ Message appears in chat (right-aligned, user bubble)
  │
  ├─ Agent status bar changes to "Working..." (amber pulse)
  │
  ├─ Agent message bubble appears (left-aligned)
  │    ├─ Streaming text renders incrementally
  │    ├─ Status chips appear inline as tools execute
  │    └─ Status feed updates in real-time
  │
  └─ Agent finishes → status returns to "Idle" (green)
```

### 5.3 Interrupt an Agent (CRITICAL FLOW)

```
Agent is working on a task (status: amber, streaming updates)
  │
  ├─ User types a new message
  │    └─ Input bar shows ⚡ Interrupt button (replaces Send)
  │
  ├─ User taps ⚡ Interrupt (or Send with text)
  │
  ├─ Frontend sends interrupt signal + new message via WebSocket
  │
  ├─ Backend:
  │    ├─ Pauses/cancels current execution loop
  │    ├─ Injects user message into agent context
  │    └─ Agent reads new message and adapts behavior
  │
  ├─ In chat:
  │    ├─ Previous task's status chip → "Interrupted" (grey)
  │    ├─ User's interrupt message appears
  │    └─ Agent begins responding to new message
  │
  └─ Agent processes new directive
```

### 5.4 Switch Between Agents

```
User is chatting with Sam
  │
  ├─ Taps Agent Selector dropdown (top-left)
  │
  ├─ Dropdown shows all agents with status indicators
  │
  ├─ User selects "Bob"
  │
  ├─ Chat view transitions (subtle crossfade):
  │    ├─ Message list loads Bob's conversation history
  │    ├─ Agent status bar updates to Bob's current state
  │    └─ Input bar adapts (shows interrupt if Bob is working)
  │
  └─ Sam continues working in background (visible on Dashboard)
```

### 5.5 Create a New Agent

```
User taps [+ New Agent] (Chat top bar or Dashboard)
  │
  ├─ "Create New Agent" bottom sheet slides up
  │
  ├─ User fills in:
  │    ├─ Name: "Debug-Bot"
  │    ├─ System Prompt: "You are a debugging assistant..."
  │    ├─ Type: Subagent
  │    ├─ Parent: Sam
  │    ├─ Tools: [Shell, Read File]
  │    └─ LLM: Use default
  │
  ├─ Taps [Create Agent]
  │
  ├─ Bottom sheet dismisses
  │
  ├─ Agent appears in:
  │    ├─ Agent Selector dropdown
  │    ├─ Dashboard agent cards
  │    └─ Chat auto-switches to new agent
  │
  └─ User can immediately start chatting
```

### 5.6 Monitor from Dashboard

```
User navigates to Dashboard tab
  │
  ├─ Sees all agents as cards with live status
  │
  ├─ Sees "Background Tasks" section with step-by-step logs
  │
  ├─ User notices Alice has an error
  │    ├─ Taps [View Chat] → navigates to Chat with Alice
  │    ├─ OR taps [Retry] → re-runs last task
  │    └─ OR taps [Dismiss] → clears error
  │
  └─ Dashboard updates in real-time via WebSocket
```

---

## 6. Interaction Patterns

### 6.1 Animations & Transitions

| Interaction              | Animation                                                    |
|--------------------------|--------------------------------------------------------------|
| Tab switch               | Crossfade (200ms ease-in-out)                                |
| Bottom sheet open        | Slide up from bottom (300ms spring curve)                    |
| Bottom sheet dismiss     | Slide down + fade (200ms)                                    |
| Message appear           | Fade in + slight upward slide (150ms)                        |
| Status chip appear       | Scale in from 0.8 to 1.0 (100ms)                            |
| Agent status pulse       | Opacity oscillation 0.6-1.0 (1.5s loop, ease-in-out)        |
| Streaming text           | Character-by-character or word-by-word render                |
| Interrupt acknowledgment | Brief screen-edge flash (amber, 100ms)                       |

### 6.2 Error States

| Error                        | UI Response                                                       |
|------------------------------|-------------------------------------------------------------------|
| Backend not running          | Banner at top: "Backend offline. [Restart]". All agent cards grey.|
| LLM endpoint unreachable     | Toast: "Cannot reach LLM endpoint. Check Settings."              |
| Agent execution error        | Status chip turns red. Error detail in expandable section.        |
| WebSocket disconnected       | Subtle amber banner: "Reconnecting..." with auto-retry.          |
| No agents configured         | Empty state illustration + "Create your first agent" CTA.        |

### 6.3 Empty States

| Screen       | Empty State                                                              |
|--------------|--------------------------------------------------------------------------|
| Chat         | Centered: Zeclaw logo (subtle), "Start by sending a message" hint text.  |
| Dashboard    | Centered: "No agents running. Create one from Chat." + [Create Agent].   |
| Settings     | Endpoints section: "No endpoints configured. Add one to get started."    |

### 6.4 Accessibility

- Minimum touch target: 48x48dp
- All interactive elements have semantic labels
- Status indicators use both color AND icon/text (not color alone)
- Support for system font scaling
- High contrast between text and background (WCAG AA minimum)
