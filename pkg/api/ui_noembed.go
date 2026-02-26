//go:build !embed_ui

package api

import (
	"net/http"

	"github.com/goclaw/goclaw/pkg/logger"
)

func newUIHandler(_ logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte("UI not included. Rebuild with -tags embed_ui"))
	})
}
