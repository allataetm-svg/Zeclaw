package server

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/allataetm-svg/zeclaw/backend/internal/agent"
	"github.com/allataetm-svg/zeclaw/backend/internal/protocol"
	"github.com/allataetm-svg/zeclaw/backend/internal/storage"
	"github.com/allataetm-svg/zeclaw/backend/internal/tools"
)

const (
	pingInterval = 30 * time.Second
	pongWait     = 60 * time.Second
	writeWait    = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow connections from any origin – the server only listens on
	// localhost so this is safe in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client represents a single connected WebSocket peer.
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

// Server manages WebSocket connections, the tool registry, and agent
// lifecycle.
type Server struct {
	mu       sync.RWMutex
	clients  map[*Client]bool
	store    *storage.Store
	agentMgr *agent.Manager
	registry *tools.Registry
}

// New constructs a Server, registers built-in tools, and loads persisted
// agents from the database.
func New(store *storage.Store, workspaceDir string) *Server {
	registry := tools.NewRegistry()
	registry.Register(tools.NewShellTool())
	registry.Register(tools.NewReadFileTool(workspaceDir))

	s := &Server{
		clients:  make(map[*Client]bool),
		store:    store,
		registry: registry,
	}

	s.agentMgr = agent.NewManager(registry, store, s.Broadcast)

	if err := s.agentMgr.LoadAgentsFromDB(); err != nil {
		log.Printf("Warning: failed to load agents from DB: %v", err)
	}

	return s
}

// Start registers HTTP handlers and begins listening on addr.
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("Zeclaw backend listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

// ---------------------------------------------------------------------------
// WebSocket upgrade & I/O pumps
// ---------------------------------------------------------------------------

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	log.Printf("New WebSocket client connected from %s", r.RemoteAddr)

	go s.writePump(client)
	s.readPump(client) // blocks until the connection closes
}

// readPump reads messages from the WebSocket connection and routes them to
// the appropriate handler.  It runs on the goroutine spawned by handleWS.
func (s *Server) readPump(client *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
		client.conn.Close()
		log.Printf("WebSocket client disconnected")
	}()

	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
		s.handleMessage(client, message)
	}
}

// writePump drains the client's send channel and forwards messages over the
// WebSocket.  It also sends periodic pings.
func (s *Server) writePump(client *Client) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed: send a close frame.
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Broadcast helpers
// ---------------------------------------------------------------------------

// Broadcast serialises env and sends it to every connected WebSocket client.
func (s *Server) Broadcast(env *protocol.Envelope) {
	data, err := env.Marshal()
	if err != nil {
		log.Printf("Failed to marshal envelope: %v", err)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients {
		select {
		case client.send <- data:
		default:
			log.Printf("Client send buffer full, dropping message")
		}
	}
}

// SendToClient serialises env and sends it only to the specified client.
func (s *Server) SendToClient(client *Client, env *protocol.Envelope) {
	data, err := env.Marshal()
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
		log.Printf("Client send buffer full for targeted send, dropping message")
	}
}
