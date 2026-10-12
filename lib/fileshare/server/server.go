package server

import (
	"net/http"
	"path/filepath"
	"strings"
)

// NewServer configures and returns the HTTP handler with all routes and middleware.
func NewServer(mediaRoot, baseDir, onionDomain string) http.Handler {
	handlers := NewHandlers(mediaRoot, baseDir, onionDomain)
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("GET /get_disk_info", handlers.HandleGetDiskInfo)
	mux.HandleFunc("GET /get_tree", handlers.HandleGetTree)
	mux.HandleFunc("POST /upload_files", handlers.HandleUploadFiles)
	mux.HandleFunc("GET /download", handlers.HandleDownload)
	mux.HandleFunc("GET /download_folder", handlers.HandleDownloadFolder)

	// Static assets from webclient/dist/assets
	assetsDir := filepath.Join(baseDir, "webclient", "dist", "assets")
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))

	// Root and SPA fallback handler
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// If path starts with /assets/ or /get_ or /upload_ or /download, let 404 happen if not matched
		if strings.HasPrefix(r.URL.Path, "/assets/") ||
			r.URL.Path == "/get_disk_info" ||
			r.URL.Path == "/get_tree" ||
			r.URL.Path == "/upload_files" ||
			r.URL.Path == "/download" ||
			r.URL.Path == "/download_folder" {
			http.NotFound(w, r)
			return
		}
		handlers.HandleIndex(w, r)
	})

	// Wrap with Gzip and CORS middleware
	handler := GzipMiddleware(mux)
	handler = CORSMiddleware(onionDomain, handler)

	return handler
}
