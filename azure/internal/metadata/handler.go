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
		"environment":    "TinyCloudLocal",
		"tenantId":       h.cfg.TenantID,
		"subscriptionId": h.cfg.SubscriptionID,
		"authentication": map[string]any{
			"issuer":          h.cfg.EffectiveTokenIssuer(),
			"audience":        h.cfg.TokenAudience,
			"oauthToken":      h.cfg.OAuthTokenURL(),
			"managedIdentity": h.cfg.ManagedIdentityURL(),
		},
		"management": map[string]string{
			"arm":      h.cfg.ManagementHTTPURL(),
			"armHttps": h.cfg.ManagementTLSURL(),
			"metadata": h.cfg.ManagementHTTPURL() + "/metadata/endpoints",
			"identity": h.cfg.ManagementHTTPURL() + "/metadata/identity",
			"oauth":    h.cfg.OAuthTokenURL(),
		},
		"resourceManager": map[string]any{
			"endpoint":    h.cfg.ManagementHTTPURL(),
			"apiVersions": []string{"2024-01-01", "2018-02-01"},
			"providers":   []string{"Microsoft.Resources", "Microsoft.Storage", "Microsoft.KeyVault"},
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
