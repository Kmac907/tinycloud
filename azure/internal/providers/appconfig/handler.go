package appconfig

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

type Handler struct {
	store *state.Store
	cfg   config.Config
}

func NewHandler(store *state.Store, cfg config.Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /stores", h.listStores)
	mux.HandleFunc("POST /stores", h.createStore)
	mux.HandleFunc("GET /stores/{store}/kv", h.listValues)
	mux.HandleFunc("PUT /stores/{store}/kv/{key}", h.putValue)
	mux.HandleFunc("GET /stores/{store}/kv/{key}", h.getValue)
	mux.HandleFunc("DELETE /stores/{store}/kv/{key}", h.deleteValue)
}

func (h *Handler) listStores(w http.ResponseWriter, _ *http.Request) {
	stores, err := h.store.ListAppConfigStores()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(stores))
	for _, store := range stores {
		value = append(value, map[string]any{
			"name": store.Name,
			"id":   h.storeID(store.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	store, created, err := h.store.CreateAppConfigStore(body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "store already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": store.Name,
		"id":   h.storeID(store.Name),
	})
}

func (h *Handler) listValues(w http.ResponseWriter, r *http.Request) {
	values, err := h.store.ListAppConfigValues(r.PathValue("store"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(values))
	for _, item := range values {
		value = append(value, valueResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putValue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Value       string `json:"value"`
		ContentType string `json:"contentType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	value, err := h.store.PutAppConfigValue(r.PathValue("store"), r.PathValue("key"), r.URL.Query().Get("label"), body.Value, body.ContentType)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, valueResponse(value))
}

func (h *Handler) getValue(w http.ResponseWriter, r *http.Request) {
	value, err := h.store.GetAppConfigValue(r.PathValue("store"), r.PathValue("key"), r.URL.Query().Get("label"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, valueResponse(value))
}

func (h *Handler) deleteValue(w http.ResponseWriter, r *http.Request) {
	err := h.store.DeleteAppConfigValue(r.PathValue("store"), r.PathValue("key"), r.URL.Query().Get("label"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) storeID(store string) string {
	return h.cfg.AppConfigURL() + "/stores/" + store
}

func valueResponse(value state.AppConfigValue) map[string]any {
	return map[string]any{
		"key":         value.Key,
		"label":       value.Label,
		"value":       value.Value,
		"contentType": value.ContentType,
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
