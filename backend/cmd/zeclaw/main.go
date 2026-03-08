package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/allataetm-svg/zeclaw/backend/internal/server"
	"github.com/allataetm-svg/zeclaw/backend/internal/storage"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8085", "WebSocket server address")
	dataDir := flag.String("data-dir", defaultDataDir(), "Data directory for SQLite and workspaces")
	flag.Parse()

	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Ensure data and workspace directories exist.
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	workspaceDir := filepath.Join(*dataDir, "workspaces")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		log.Fatalf("Failed to create workspace directory: %v", err)
	}

	// Initialise storage.
	dbPath := filepath.Join(*dataDir, "zeclaw.db")
	store, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialise storage: %v", err)
	}
	defer store.Close()

	log.Printf("Zeclaw backend starting...")
	log.Printf("Data directory : %s", *dataDir)
	log.Printf("Database       : %s", dbPath)
	log.Printf("Listen address : %s", *addr)

	// Start WebSocket server (blocks until the process exits).
	srv := server.New(store, workspaceDir)
	if err := srv.Start(*addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// defaultDataDir returns ~/.zeclaw or /tmp/zeclaw if the home directory
// cannot be determined.
func defaultDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/zeclaw"
	}
	return filepath.Join(homeDir, ".zeclaw")
}
