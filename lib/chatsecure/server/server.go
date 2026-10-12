package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"torbox/chatsecure/storage"
)

type ServerConfig struct {
	BaseDir     string
	OnionDomain string
	SecretKey   string
	Dev         bool
	Debug       bool
}

// CORSMiddleware handles CORS headers and preflight OPTIONS requests.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests if debug is enabled.
func LoggingMiddleware(debug bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if debug && !strings.HasPrefix(r.URL.Path, "/assets/") {
			log.Printf("[http] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}
		next.ServeHTTP(w, r)
	})
}

// NewServer configures and returns the HTTP handler with all routes and middleware.
func NewServer(cfg ServerConfig, s *storage.Storage, hub *Hub) http.Handler {
	api := NewAPIHandler(s, hub, cfg.SecretKey, cfg.Debug)
	mux := http.NewServeMux()

	// Authentication
	mux.HandleFunc("/login", api.HandleLogin)
	mux.HandleFunc("/api/login", api.HandleLogin)

	// User directory
	mux.HandleFunc("/users", api.HandleUsers)
	mux.HandleFunc("/api/users", api.HandleUsers)

	// Groups
	mux.HandleFunc("/groups", api.HandleGroups)
	mux.HandleFunc("/api/groups", api.HandleGroups)

	// Messages - Hybrid resilient endpoints
	mux.HandleFunc("/api/messages/send", api.HandleSendMessage)
	mux.HandleFunc("/api/messages/sync", api.HandleSyncMessages)

	// Historical messages
	mux.HandleFunc("/group_msg", api.HandleGroupMessages)
	mux.HandleFunc("/api/messages/group/default", api.HandleGroupMessages)

	// User message history routing
	userMsgHandler := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		var senderIDStr string
		if strings.HasPrefix(path, "/user_msg/") {
			senderIDStr = strings.TrimPrefix(path, "/user_msg/")
		} else if strings.HasPrefix(path, "/api/messages/user/") {
			senderIDStr = strings.TrimPrefix(path, "/api/messages/user/")
		}
		senderIDStr = strings.Trim(senderIDStr, "/")
		if senderIDStr == "" {
			http.Error(w, "Sender ID required", http.StatusBadRequest)
			return
		}
		api.HandleUserMessages(w, r, senderIDStr)
	}

	mux.HandleFunc("/user_msg/", userMsgHandler)
	mux.HandleFunc("/api/messages/user/", userMsgHandler)

	// WebSocket & Socket.IO
	mux.HandleFunc("/ws", hub.ServeWs)
	mux.HandleFunc("/socket.io/", hub.ServeSocketIO)

	// Locate static assets directory (check webclient/dist/assets, then chatsecure/templates/assets)
	assetsDir := filepath.Join(cfg.BaseDir, "webclient", "dist", "assets")
	if fi, err := os.Stat(assetsDir); err != nil || !fi.IsDir() {
		assetsDir = filepath.Join(cfg.BaseDir, "chatsecure", "templates", "assets")
	}

	indexFile := filepath.Join(cfg.BaseDir, "webclient", "dist", "index.html")
	if _, err := os.Stat(indexFile); err != nil {
		indexFile = filepath.Join(cfg.BaseDir, "chatsecure", "templates", "index.html")
	}

	// Serve assets
	assetsFS := http.FileServer(http.Dir(assetsDir))
	mux.Handle("/assets/", http.StripPrefix("/assets/", assetsFS))
	mux.Handle("/static/", http.StripPrefix("/static/", assetsFS))

	// Serve index.html and SPA fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// If it looks like an API call or socket request that didn't match, 404
			if strings.HasPrefix(r.URL.Path, "/api/") ||
				strings.HasPrefix(r.URL.Path, "/user_msg") ||
				strings.HasPrefix(r.URL.Path, "/group_msg") ||
				strings.HasPrefix(r.URL.Path, "/login") ||
				strings.HasPrefix(r.URL.Path, "/users") ||
				strings.HasPrefix(r.URL.Path, "/groups") ||
				strings.HasPrefix(r.URL.Path, "/ws") ||
				strings.HasPrefix(r.URL.Path, "/socket.io") {
				http.NotFound(w, r)
				return
			}
		}
		http.ServeFile(w, r, indexFile)
	})

	handler := CORSMiddleware(mux)
	handler = LoggingMiddleware(cfg.Debug, handler)
	return handler
}
