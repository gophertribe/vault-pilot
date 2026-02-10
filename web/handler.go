package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

// MIME types for common asset extensions
var mimeTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".json":  "application/json",
	".ico":   "image/x-icon",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".woff":  "font/woff",
	".woff2": "font/woff2",
}

// NewHandler returns an http.Handler that serves the embedded frontend.
// - /app/assets/* → static files from dist/assets
// - /app/* → SPA fallback (index.html) for client-side routing
// - /app → redirect to /app/
func NewHandler(assets embed.FS) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/app" {
			http.Redirect(w, r, "/app/", http.StatusMovedPermanently)
			return
		}

		// Strip /app prefix to get the path within dist
		reqPath := strings.TrimPrefix(r.URL.Path, "/app")
		if reqPath == "" {
			reqPath = "/"
		}
		// Prevent path traversal
		if strings.Contains(reqPath, "..") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// Try to serve static file first
		filePath := "dist" + reqPath
		if reqPath == "/" {
			filePath = "dist/index.html"
		}

		data, err := fs.ReadFile(assets, filePath)
		if err == nil {
			// Serve with appropriate content-type and cache headers for assets
			ext := path.Ext(reqPath)
			if ct, ok := mimeTypes[ext]; ok {
				w.Header().Set("Content-Type", ct)
			}
			if ext == ".js" || ext == ".css" {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}

		// SPA fallback: serve index.html for any /app/* path
		data, err = fs.ReadFile(assets, "dist/index.html")
		if err != nil {
			log.Printf("web: failed to read index.html: %v", err)
			http.Error(w, "frontend not available", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}), nil
}
