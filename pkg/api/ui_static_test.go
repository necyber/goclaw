package api

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestEmbeddedUIHandler_SPAFallback(t *testing.T) {
	handler := newEmbeddedUIHandler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/workflows/abc-123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "<html>ui</html>") {
		t.Fatalf("expected SPA fallback to index.html, got %q", rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-cache")
	}
}

func TestEmbeddedUIHandler_StaticAssetHeaders(t *testing.T) {
	js := strings.Repeat("a", 2048)
	handler := newEmbeddedUIHandler(fstest.MapFS{
		"index.html":                 &fstest.MapFile{Data: []byte("<html>ui</html>")},
		"assets/main.a1b2c3d4.js":    &fstest.MapFile{Data: []byte(js)},
		"assets/small.abc12345.json": &fstest.MapFile{Data: []byte("{}")},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/assets/main.a1b2c3d4.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable cache header", got)
	}

	gz, err := gzip.NewReader(strings.NewReader(rec.Body.String()))
	if err != nil {
		t.Fatalf("failed to read gzip body: %v", err)
	}
	defer gz.Close()
	plain, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to read uncompressed body: %v", err)
	}
	if string(plain) != js {
		t.Fatalf("unexpected gzip body content")
	}
}

func TestEmbeddedUIHandler_MissingAssetReturnsNotFound(t *testing.T) {
	handler := newEmbeddedUIHandler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>ui</html>")},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
