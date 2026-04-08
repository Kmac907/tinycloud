package table

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
	mux.HandleFunc("GET /{account}/Tables", h.listTables)
	mux.HandleFunc("POST /{account}/Tables", h.createTable)
	mux.HandleFunc("DELETE /{account}/Tables/{table}", h.deleteTable)
	mux.HandleFunc("GET /{account}/{table}", h.listEntities)
	mux.HandleFunc("POST /{account}/{table}", h.upsertEntity)
	mux.HandleFunc("GET /{account}/{table}/{partitionKey}/{rowKey}", h.getEntity)
	mux.HandleFunc("DELETE /{account}/{table}/{partitionKey}/{rowKey}", h.deleteEntity)
}

func (h *Handler) listTables(w http.ResponseWriter, r *http.Request) {
	if !h.accountExists(w, r, r.PathValue("account")) {
		return
	}

	tables, err := h.store.ListTables(r.PathValue("account"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(tables))
	for _, table := range tables {
		value = append(value, map[string]any{
			"TableName": table.Name,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createTable(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TableName string `json:"TableName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.TableName == "" {
		http.Error(w, "TableName is required", http.StatusBadRequest)
		return
	}

	table, created, err := h.store.CreateTable(r.PathValue("account"), body.TableName)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "table already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"TableName": table.Name,
	})
}

func (h *Handler) deleteTable(w http.ResponseWriter, r *http.Request) {
	err := h.store.DeleteTable(r.PathValue("account"), r.PathValue("table"))
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

func (h *Handler) listEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := h.store.ListTableEntities(r.PathValue("account"), r.PathValue("table"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(entities))
	for _, entity := range entities {
		value = append(value, entityResponse(entity))
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) upsertEntity(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	partitionKey, _ := body["PartitionKey"].(string)
	rowKey, _ := body["RowKey"].(string)
	if partitionKey == "" || rowKey == "" {
		http.Error(w, "PartitionKey and RowKey are required", http.StatusBadRequest)
		return
	}

	delete(body, "PartitionKey")
	delete(body, "RowKey")

	entity, err := h.store.UpsertTableEntity(r.PathValue("account"), r.PathValue("table"), partitionKey, rowKey, body)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, entityResponse(entity))
}

func (h *Handler) getEntity(w http.ResponseWriter, r *http.Request) {
	entity, err := h.store.GetTableEntity(r.PathValue("account"), r.PathValue("table"), r.PathValue("partitionKey"), r.PathValue("rowKey"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, entityResponse(entity))
}

func (h *Handler) deleteEntity(w http.ResponseWriter, r *http.Request) {
	err := h.store.DeleteTableEntity(r.PathValue("account"), r.PathValue("table"), r.PathValue("partitionKey"), r.PathValue("rowKey"))
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

func (h *Handler) accountExists(w http.ResponseWriter, r *http.Request, accountName string) bool {
	if _, err := h.store.GetStorageAccountByName(accountName); errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return false
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func entityResponse(entity state.TableEntity) map[string]any {
	body := map[string]any{
		"PartitionKey": entity.PartitionKey,
		"RowKey":       entity.RowKey,
	}
	for key, value := range entity.Properties {
		body[key] = value
	}
	return body
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
