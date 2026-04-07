package admin

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tinycloud/internal/httpx"
	"tinycloud/internal/state"
)

type Handler struct {
	store    *state.Store
	dataRoot string
}

func NewHandler(store *state.Store, dataRoot string) *Handler {
	return &Handler{store: store, dataRoot: dataRoot}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /_admin/healthz", h.health)
	mux.HandleFunc("GET /_admin/metrics", h.metrics)
	mux.HandleFunc("POST /_admin/reset", h.reset)
	mux.HandleFunc("POST /_admin/snapshot", h.snapshot)
	mux.HandleFunc("POST /_admin/seed", h.seed)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) metrics(w http.ResponseWriter, _ *http.Request) {
	summary, err := h.store.Summary()
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"resourceCount": summary.ResourceCount,
		"updatedAt":     summary.UpdatedAt,
		"statePath":     summary.StatePath,
	})
}

func (h *Handler) reset(w http.ResponseWriter, _ *http.Request) {
	if err := h.store.Reset(); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) {
	path, err := h.resolveDataPath(r.URL.Query().Get("path"), "tinycloud.snapshot.json")
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.Snapshot(path); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"snapshot": path})
}

func (h *Handler) seed(w http.ResponseWriter, r *http.Request) {
	queryPath := r.URL.Query().Get("path")
	if queryPath == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing path query parameter"})
		return
	}
	path, err := h.resolveDataPath(queryPath, "")
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.ApplySeed(path); err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"seed": path})
}

func (h *Handler) resolveDataPath(queryPath, fallback string) (string, error) {
	root, err := filepath.Abs(h.dataRoot)
	if err != nil {
		return "", errors.New("resolve data root")
	}

	target := queryPath
	if target == "" {
		target = fallback
	}
	if target == "" {
		return "", errors.New("missing path query parameter")
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}

	target, err = filepath.Abs(target)
	if err != nil {
		return "", errors.New("resolve path")
	}

	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", errors.New("resolve path")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", errors.New("path must stay within data root")
	}

	return target, nil
}
