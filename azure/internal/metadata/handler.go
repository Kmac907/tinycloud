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
		"name":           "TinyCloud",
		"tenantId":       h.cfg.TenantID,
		"subscriptionId": h.cfg.SubscriptionID,
		"management": map[string]string{
			"arm":      h.cfg.ManagementHTTPURL(),
			"armHttps": h.cfg.ManagementTLSURL(),
			"metadata": h.cfg.ManagementHTTPURL() + "/metadata/endpoints",
			"identity": h.cfg.ManagementHTTPURL() + "/metadata/identity",
			"oauth":    h.cfg.OAuthTokenURL(),
		},
		"services": map[string]string{
			"blob":       h.cfg.BlobURL(),
			"queue":      h.cfg.QueueURL(),
			"table":      h.cfg.TableURL(),
			"keyVault":   h.cfg.KeyVaultURL(),
			"serviceBus": h.cfg.ServiceBusURL(),
		},
	})
}
