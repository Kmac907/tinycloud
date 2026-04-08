package keyvault

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

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
	mux.HandleFunc("GET /{vault}/secrets", h.listSecrets)
	mux.HandleFunc("PUT /{vault}/secrets/{secretName}", h.putSecret)
	mux.HandleFunc("GET /{vault}/secrets/{secretName}", h.getSecret)
	mux.HandleFunc("DELETE /{vault}/secrets/{secretName}", h.deleteSecret)
}

func (h *Handler) listSecrets(w http.ResponseWriter, r *http.Request) {
	vaultName := r.PathValue("vault")
	if _, err := h.store.GetKeyVaultByName(vaultName); errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	secrets, err := h.store.ListKeyVaultSecrets(vaultName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(secrets))
	for _, secret := range secrets {
		value = append(value, h.secretListResponse(secret))
	}

	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) putSecret(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Value       string `json:"value"`
		ContentType string `json:"contentType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	secret, err := h.store.PutKeyVaultSecret(r.PathValue("vault"), r.PathValue("secretName"), body.Value, body.ContentType)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, h.secretResponse(secret, true))
}

func (h *Handler) getSecret(w http.ResponseWriter, r *http.Request) {
	secret, err := h.store.GetKeyVaultSecret(r.PathValue("vault"), r.PathValue("secretName"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, h.secretResponse(secret, true))
}

func (h *Handler) deleteSecret(w http.ResponseWriter, r *http.Request) {
	secret, err := h.store.GetKeyVaultSecret(r.PathValue("vault"), r.PathValue("secretName"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.DeleteKeyVaultSecret(r.PathValue("vault"), r.PathValue("secretName")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, h.secretResponse(secret, true))
}

func (h *Handler) secretResponse(secret state.KeyVaultSecret, includeValue bool) map[string]any {
	body := map[string]any{
		"id":          h.secretID(secret.VaultName, secret.Name),
		"contentType": secret.ContentType,
		"attributes": map[string]any{
			"created": toUnixSeconds(secret.CreatedAt),
			"updated": toUnixSeconds(secret.UpdatedAt),
			"enabled": true,
		},
	}
	if includeValue {
		body["value"] = secret.Value
	}
	return body
}

func (h *Handler) secretListResponse(secret state.KeyVaultSecret) map[string]any {
	return h.secretResponse(secret, false)
}

func (h *Handler) secretID(vaultName, secretName string) string {
	return h.cfg.KeyVaultURL() + "/" + vaultName + "/secrets/" + secretName
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func toUnixSeconds(value string) int64 {
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0
	}
	return timestamp.UTC().Unix()
}
