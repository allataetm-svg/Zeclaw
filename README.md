# Zeclaw

Zeclaw is a mobile-first AI agent platform that runs entirely on-device. It ships a compiled Go binary alongside a Flutter front-end so that an autonomous coding / tool-use agent operates without any cloud dependency.

---

## Features

- **On-device AI agent** – the Go backend embeds an LLM client and tool-execution engine; no data leaves the phone.
- **WebSocket-based IPC** – the Flutter UI communicates with the local backend over a loopback WebSocket, giving a clean protocol boundary.
- **Pluggable tools** – file system, shell, web-fetch, and custom tools are registered in `backend/internal/tools/`.
- **Streaming responses** – the agent streams tokens back to the UI in real time using server-sent events over the WebSocket connection.
- **Offline-first** – conversation history is persisted in a local SQLite database via `backend/internal/storage/`.
- **Material You theming** – the Flutter front-end adopts dynamic color derived from the device wallpaper on Android 12+.

---

## Repository Layout

```
zeclaw/
├── docs/                   # Architecture, UI/UX, and implementation docs
├── frontend/               # Flutter application (Android target)
│   ├── lib/
│   │   ├── main.dart
│   │   ├── app/            # App-level routing and initialization
│   │   ├── features/       # Feature modules (chat, dashboard, settings)
│   │   ├── core/           # Shared utilities (websocket, models, providers, theme)
│   │   └── widgets/        # Reusable UI components
│   ├── pubspec.yaml
│   └── test/
├── backend/                # Go backend (compiled to Android binary)
│   ├── cmd/zeclaw/         # Entry point (main.go)
│   ├── internal/
│   │   ├── server/         # WebSocket server and HTTP handlers
│   │   ├── agent/          # Core agent loop and orchestration
│   │   ├── tools/          # Tool definitions and execution
│   │   ├── llm/            # LLM provider clients
│   │   ├── storage/        # SQLite persistence layer
│   │   └── protocol/       # Message types shared between UI and backend
│   ├── go.mod
│   └── Makefile
├── scripts/
│   ├── build-backend.sh    # Cross-compile Go for Android ARM64/ARM
│   └── run-dev.sh          # Run backend locally for development
└── .github/workflows/
    └── ci.yml              # Go + Flutter CI pipeline
```

---

## Prerequisites

| Tool | Version |
|------|---------|
| Go | ≥ 1.22 |
| Flutter | stable channel |
| Android SDK | API 24+ |
| Android NDK | (not required – CGO is disabled) |

---

## Setup

### 1. Clone the repo

```bash
git clone https://github.com/your-org/zeclaw.git
cd zeclaw
```

### 2. Build the Go backend for Android

```bash
chmod +x scripts/build-backend.sh
./scripts/build-backend.sh
```

This produces two binaries under `frontend/assets/backend/`:

- `zeclaw-backend-arm64` – for 64-bit Android devices
- `zeclaw-backend-arm`   – for 32-bit Android devices

### 3. Install Flutter dependencies

```bash
cd frontend
flutter pub get
```

### 4. Run on a connected device / emulator

```bash
cd frontend
flutter run
```

### Development (backend only)

```bash
chmod +x scripts/run-dev.sh
./scripts/run-dev.sh
```

The backend starts on `localhost:7777` by default.

---

## Architecture Overview

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full system design.

The high-level data flow is:

```
Flutter UI  ←──WebSocket──→  Go Backend  ←──HTTP/gRPC──→  LLM Provider
                                  │
                             Tool Executor
                                  │
                       (filesystem, shell, web, …)
```

---

## Contributing

1. Fork the repo and create a branch from `main`.
2. Run `go vet ./...` and `go test ./...` inside `backend/` before pushing.
3. Run `flutter analyze` and `flutter test` inside `frontend/` before pushing.
4. Open a pull request – CI will validate both automatically.

---

## License

MIT – see `LICENSE` for details.
