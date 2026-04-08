package eventhubs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

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
	mux.HandleFunc("GET /namespaces", h.listNamespaces)
	mux.HandleFunc("POST /namespaces", h.createNamespace)
	mux.HandleFunc("GET /namespaces/{namespace}/hubs", h.listHubs)
	mux.HandleFunc("POST /namespaces/{namespace}/hubs", h.createHub)
	mux.HandleFunc("POST /namespaces/{namespace}/hubs/{hub}/events", h.publishEvent)
	mux.HandleFunc("GET /namespaces/{namespace}/hubs/{hub}/events", h.listEvents)
}

func (h *Handler) listNamespaces(w http.ResponseWriter, _ *http.Request) {
	namespaces, err := h.store.ListEventHubNamespaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(namespaces))
	for _, namespace := range namespaces {
		value = append(value, map[string]any{
			"name": namespace.Name,
			"id":   h.namespaceID(namespace.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createNamespace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	namespace, created, err := h.store.CreateEventHubNamespace(body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "namespace already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": namespace.Name,
		"id":   h.namespaceID(namespace.Name),
	})
}

func (h *Handler) listHubs(w http.ResponseWriter, r *http.Request) {
	hubs, err := h.store.ListEventHubs(r.PathValue("namespace"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(hubs))
	for _, hub := range hubs {
		value = append(value, map[string]any{
			"name": hub.Name,
			"id":   h.hubID(hub.Namespace, hub.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createHub(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	hub, created, err := h.store.CreateEventHub(r.PathValue("namespace"), body.Name)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "event hub already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": hub.Name,
		"id":   h.hubID(hub.Namespace, hub.Name),
	})
}

func (h *Handler) publishEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Body         string `json:"body"`
		PartitionKey string `json:"partitionKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	event, err := h.store.PublishEventHubEvent(r.PathValue("namespace"), r.PathValue("hub"), body.Body, body.PartitionKey)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, eventResponse(event))
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	fromSequenceNumber, maxEvents, ok := listOptions(w, r)
	if !ok {
		return
	}

	events, err := h.store.ListEventHubEvents(r.PathValue("namespace"), r.PathValue("hub"), fromSequenceNumber, maxEvents)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(events))
	for _, event := range events {
		value = append(value, eventResponse(event))
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) namespaceID(namespace string) string {
	return h.cfg.EventHubsURL() + "/namespaces/" + namespace
}

func (h *Handler) hubID(namespace, hub string) string {
	return h.namespaceID(namespace) + "/hubs/" + hub
}

func listOptions(w http.ResponseWriter, r *http.Request) (int64, int, bool) {
	var fromSequenceNumber int64
	if raw := strings.TrimSpace(r.URL.Query().Get("fromSequenceNumber")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 0 {
			http.Error(w, "fromSequenceNumber must be a non-negative integer", http.StatusBadRequest)
			return 0, 0, false
		}
		fromSequenceNumber = value
	}

	maxEvents := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("maxEvents")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			http.Error(w, "maxEvents must be a positive integer", http.StatusBadRequest)
			return 0, 0, false
		}
		maxEvents = value
	}

	return fromSequenceNumber, maxEvents, true
}

func eventResponse(event state.EventHubEvent) map[string]any {
	return map[string]any{
		"id":             event.ID,
		"body":           event.Body,
		"partitionKey":   event.PartitionKey,
		"sequenceNumber": event.SequenceNumber,
		"offset":         event.Offset,
		"enqueuedTime":   event.EnqueuedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
