package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mqverk/shlx/backend/pty"
	"github.com/mqverk/shlx/backend/session"
	"github.com/mqverk/shlx/backend/websocket"
)

var (
	port    = flag.String("port", "8080", "Server port")
	host    = flag.String("host", "localhost", "Server host")
	shell   = flag.String("shell", "", "Shell to use (default: $SHELL or /bin/bash)")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	manager := session.NewManager()

	// HTTP routes
	http.HandleFunc("/api/create", createSessionHandler(manager))
	http.HandleFunc("/api/session/", sessionInfoHandler(manager))
	http.HandleFunc("/ws", websocket.NewHandler(manager).HandleConnection)
	http.HandleFunc("/health", healthHandler)

	// Serve frontend in production
	if _, err := os.Stat("./frontend/build"); err == nil {
		fs := http.FileServer(http.Dir("./frontend/build"))
		http.Handle("/", fs)
	} else {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>shlx</title></head>
<body>
<h1>shlx - ShellX</h1>
<p>Backend is running. Please build and serve the frontend.</p>
<p>Create a session: <code>curl http://%s:%s/api/create</code></p>
</body>
</html>`, *host, *port)
		})
	}

	addr := fmt.Sprintf("%s:%s", *host, *port)
	log.Printf("shlx server starting on http://%s", addr)
	log.Printf("Create session: curl http://%s/api/create", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func createSessionHandler(manager *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Create PTY
		terminal, err := pty.New(*shell, []string{})
		if err != nil {
			log.Printf("Failed to create PTY: %v", err)
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// Create session
		sess := manager.CreateSession(terminal)

		// Start PTY reader
		go websocket.PTYReader(sess)

		response := map[string]string{
			"sessionId": sess.ID,
			"ownerToken": sess.Owner,
			"url":        fmt.Sprintf("http://%s:%s/session/%s", *host, *port, sess.ID),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		if *verbose {
			log.Printf("Created session: %s (owner: %s)", sess.ID, sess.Owner)
		}
	}
}

func sessionInfoHandler(manager *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Path[len("/api/session/"):]
		
		sess, ok := manager.GetSession(sessionID)
		if !ok {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		info := map[string]interface{}{
			"sessionId": sess.ID,
			"users":     sess.GetUsers(),
			"createdAt": sess.CreatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
