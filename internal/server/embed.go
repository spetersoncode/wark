package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:ui_placeholder
var uiFS embed.FS

// handleStatic serves embedded frontend files.
// Falls back to index.html for SPA routing.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Get the path from the URL
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Remove leading slash for fs.FS
	fsPath := strings.TrimPrefix(path, "/")

	// Try to get the embedded filesystem
	subFS, err := fs.Sub(uiFS, "ui_placeholder")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load UI files")
		return
	}

	// Try to open the file
	file, err := subFS.Open(fsPath)
	if err != nil {
		// For SPA routing, serve index.html for any unknown paths
		// (except for API routes and known static extensions)
		if !strings.HasPrefix(r.URL.Path, "/api/") &&
			!strings.HasSuffix(fsPath, ".js") &&
			!strings.HasSuffix(fsPath, ".css") &&
			!strings.HasSuffix(fsPath, ".ico") &&
			!strings.HasSuffix(fsPath, ".png") &&
			!strings.HasSuffix(fsPath, ".svg") &&
			!strings.HasSuffix(fsPath, ".woff") &&
			!strings.HasSuffix(fsPath, ".woff2") {
			// Serve index.html for SPA routes
			fsPath = "index.html"
			file, err = subFS.Open(fsPath)
			if err != nil {
				writeError(w, http.StatusNotFound, "UI not available")
				return
			}
		} else {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
	}
	defer file.Close()

	// Get file info for content type
	stat, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to stat file")
		return
	}

	// If it's a directory, serve index.html
	if stat.IsDir() {
		fsPath = strings.TrimSuffix(fsPath, "/") + "/index.html"
		file.Close()
		file, err = subFS.Open(fsPath)
		if err != nil {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		defer file.Close()
		stat, _ = file.Stat()
	}

	// Set content type based on extension
	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(fsPath, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(fsPath, ".css"):
		contentType = "text/css; charset=utf-8"
	case strings.HasSuffix(fsPath, ".js"):
		contentType = "application/javascript; charset=utf-8"
	case strings.HasSuffix(fsPath, ".json"):
		contentType = "application/json; charset=utf-8"
	case strings.HasSuffix(fsPath, ".svg"):
		contentType = "image/svg+xml"
	case strings.HasSuffix(fsPath, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(fsPath, ".ico"):
		contentType = "image/x-icon"
	case strings.HasSuffix(fsPath, ".woff"):
		contentType = "font/woff"
	case strings.HasSuffix(fsPath, ".woff2"):
		contentType = "font/woff2"
	}

	w.Header().Set("Content-Type", contentType)

	// Read file content
	if readSeeker, ok := file.(fs.File); ok {
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), readSeeker.(interface {
			Read([]byte) (int, error)
			Seek(int64, int) (int64, error)
		}))
	}
}
