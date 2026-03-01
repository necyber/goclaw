package api

import (
	_ "embed"
	"net/http"
)

var (
	//go:embed openapi/openapi.yaml
	openAPISpecYAML []byte
)

func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpecYAML)
}
