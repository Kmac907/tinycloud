package metadata

import (
	"net/http"

	"tinycloud/internal/config"
	"tinycloud/internal/httpx"
)

type Handler struct {
	cfg config.Config
}

func NewHandler(cfg config.Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /metadata/endpoints", h.endpoints)
}

func (h *Handler) endpoints(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"name":       "TinyCloud",
		"management": h.cfg.ManagementHTTPURL(),
	})
}
