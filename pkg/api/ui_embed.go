//go:build embed_ui

package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/goclaw/goclaw/pkg/logger"
)

//go:embed web/dist/**
var embeddedUIDist embed.FS

func newUIHandler(log logger.Logger) http.Handler {
	distFS, err := fs.Sub(embeddedUIDist, "web/dist")
	if err != nil {
		if log != nil {
			log.Error("failed to initialize embedded ui assets", "error", err)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "embedded UI assets are unavailable", http.StatusInternalServerError)
		})
	}
	return newEmbeddedUIHandler(distFS, log)
}
