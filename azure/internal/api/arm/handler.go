package arm

import (
	"fmt"
	"net/http"

	"tinycloud/internal/httpx"
	"tinycloud/internal/state"
)

type Handler struct {
	store *state.Store
}

func NewHandler(store *state.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /subscriptions", h.listSubscriptions)
	mux.HandleFunc("GET /providers", h.listProviders)
	mux.HandleFunc("GET /subscriptions/{subscriptionId}/resourceGroups", h.unsupportedResourceGroups)
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := h.store.ListSubscriptions()
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		value = append(value, map[string]any{
			"id":             fmt.Sprintf("/subscriptions/%s", subscription.ID),
			"subscriptionId": subscription.ID,
			"tenantId":       subscription.TenantID,
			"displayName":    "TinyCloud Local Subscription",
			"state":          "Enabled",
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) listProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.store.ListProviders()
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}

	value := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		value = append(value, map[string]any{
			"namespace":         provider.Namespace,
			"registrationState": provider.RegistrationState,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) unsupportedResourceGroups(w http.ResponseWriter, r *http.Request) {
	httpx.WriteCloudError(w, http.StatusNotImplemented, "UnsupportedOperation", "resource group routes are not implemented yet")
}
