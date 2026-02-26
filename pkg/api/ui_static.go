package api

import (
	"bytes"
	"compress/gzip"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/goclaw/goclaw/pkg/logger"
)

var uiHashedAssetPattern = regexp.MustCompile(`\.[a-fA-F0-9]{6,}\.`)

func newEmbeddedUIHandler(uiFS fs.FS, log logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		filePath, ok := resolveUIPath(uiFS, r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}

		content, modTime, err := readUIFile(uiFS, filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		contentType := mime.TypeByExtension(path.Ext(filePath))
		if contentType == "" {
			contentType = http.DetectContentType(content)
		}
		w.Header().Set("Content-Type", contentType)
		setUICacheControlHeader(w, filePath)

		if shouldGzipUIResponse(r, filePath, len(content)) {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}

			gz := gzip.NewWriter(w)
			defer func() {
				if err := gz.Close(); err != nil && log != nil {
					log.Warn("failed to close gzip writer", "error", err)
				}
			}()
			if _, err := gz.Write(content); err != nil && log != nil {
				log.Warn("failed to write gzip response", "error", err)
			}
			return
		}

		http.ServeContent(w, r, filePath, modTime, bytes.NewReader(content))
	})
}

func resolveUIPath(uiFS fs.FS, requestPath string) (string, bool) {
	cleanPath := path.Clean("/" + strings.TrimSpace(requestPath))
	if cleanPath == "/" {
		return "index.html", uiFileExists(uiFS, "index.html")
	}

	candidate := strings.TrimPrefix(cleanPath, "/")
	if uiFileExists(uiFS, candidate) {
		return candidate, true
	}

	// Treat extension-bearing paths as static assets and return 404 on miss.
	if path.Ext(candidate) != "" {
		return "", false
	}

	// SPA fallback for client-side routes.
	return "index.html", uiFileExists(uiFS, "index.html")
}

func uiFileExists(uiFS fs.FS, filePath string) bool {
	info, err := fs.Stat(uiFS, filePath)
	return err == nil && !info.IsDir()
}

func readUIFile(uiFS fs.FS, filePath string) ([]byte, time.Time, error) {
	content, err := fs.ReadFile(uiFS, filePath)
	if err != nil {
		return nil, time.Time{}, err
	}

	info, err := fs.Stat(uiFS, filePath)
	if err != nil {
		return nil, time.Time{}, err
	}

	return content, info.ModTime(), nil
}

func setUICacheControlHeader(w http.ResponseWriter, filePath string) {
	base := path.Base(filePath)
	switch {
	case base == "index.html":
		w.Header().Set("Cache-Control", "no-cache")
	case uiHashedAssetPattern.MatchString(base):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	default:
		w.Header().Set("Cache-Control", "public, max-age=300")
	}
}

func shouldGzipUIResponse(r *http.Request, filePath string, size int) bool {
	if size < 1024 {
		return false
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Accept-Encoding")), "gzip") {
		return false
	}

	switch strings.ToLower(path.Ext(filePath)) {
	case ".html", ".css", ".js", ".mjs", ".json", ".svg":
		return true
	default:
		return false
	}
}
