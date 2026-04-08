package cosmos

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
	mux.HandleFunc("GET /accounts", h.listAccounts)
	mux.HandleFunc("POST /accounts", h.createAccount)
	mux.HandleFunc("GET /accounts/{account}/dbs", h.listDatabases)
	mux.HandleFunc("POST /accounts/{account}/dbs", h.createDatabase)
	mux.HandleFunc("GET /accounts/{account}/dbs/{database}/colls", h.listContainers)
	mux.HandleFunc("POST /accounts/{account}/dbs/{database}/colls", h.createContainer)
	mux.HandleFunc("GET /accounts/{account}/dbs/{database}/colls/{container}/docs", h.listDocuments)
	mux.HandleFunc("POST /accounts/{account}/dbs/{database}/colls/{container}/docs", h.upsertDocument)
	mux.HandleFunc("GET /accounts/{account}/dbs/{database}/colls/{container}/docs/{id}", h.getDocument)
	mux.HandleFunc("DELETE /accounts/{account}/dbs/{database}/colls/{container}/docs/{id}", h.deleteDocument)
}

func (h *Handler) listAccounts(w http.ResponseWriter, _ *http.Request) {
	accounts, err := h.store.ListCosmosAccounts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(accounts))
	for _, account := range accounts {
		value = append(value, map[string]any{
			"name": account.Name,
			"id":   h.accountID(account.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
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

	account, created, err := h.store.CreateCosmosAccount(body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "account already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": account.Name,
		"id":   h.accountID(account.Name),
	})
}

func (h *Handler) listDatabases(w http.ResponseWriter, r *http.Request) {
	databases, err := h.store.ListCosmosDatabases(r.PathValue("account"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(databases))
	for _, database := range databases {
		value = append(value, map[string]any{
			"name": database.Name,
			"id":   h.databaseID(database.AccountName, database.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createDatabase(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	database, created, err := h.store.CreateCosmosDatabase(r.PathValue("account"), body.ID)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "database already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":   database.Name,
		"name": database.Name,
	})
}

func (h *Handler) listContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := h.store.ListCosmosContainers(r.PathValue("account"), r.PathValue("database"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(containers))
	for _, container := range containers {
		value = append(value, map[string]any{
			"id":               container.Name,
			"name":             container.Name,
			"partitionKeyPath": container.PartitionKeyPath,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createContainer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID               string `json:"id"`
		PartitionKeyPath string `json:"partitionKeyPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	container, created, err := h.store.CreateCosmosContainer(r.PathValue("account"), r.PathValue("database"), body.ID, body.PartitionKeyPath)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "container already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":               container.Name,
		"name":             container.Name,
		"partitionKeyPath": container.PartitionKeyPath,
	})
}

func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	documents, err := h.store.ListCosmosDocuments(r.PathValue("account"), r.PathValue("database"), r.PathValue("container"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		value = append(value, documentResponse(document))
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) upsertDocument(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	id, _ := body["id"].(string)
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	partitionKey, _ := body["partitionKey"].(string)

	document, err := h.store.UpsertCosmosDocument(r.PathValue("account"), r.PathValue("database"), r.PathValue("container"), id, partitionKey, body)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, documentResponse(document))
}

func (h *Handler) getDocument(w http.ResponseWriter, r *http.Request) {
	document, err := h.store.GetCosmosDocument(r.PathValue("account"), r.PathValue("database"), r.PathValue("container"), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, documentResponse(document))
}

func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
	err := h.store.DeleteCosmosDocument(r.PathValue("account"), r.PathValue("database"), r.PathValue("container"), r.PathValue("id"))
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

func (h *Handler) accountID(account string) string {
	return h.cfg.CosmosURL() + "/accounts/" + account
}

func (h *Handler) databaseID(account, database string) string {
	return h.accountID(account) + "/dbs/" + database
}

func documentResponse(document state.CosmosDocument) map[string]any {
	body := map[string]any{
		"id":           document.ID,
		"partitionKey": document.PartitionKey,
	}
	for key, value := range document.Body {
		body[key] = value
	}
	return body
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
