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
	managementURL := h.cfg.ManagementHTTPURL()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"name":           "TinyCloud",
		"environment":    "TinyCloudLocal",
		"tenantId":       h.cfg.TenantID,
		"subscriptionId": h.cfg.SubscriptionID,
		"authentication": map[string]any{
			"issuer":                     h.cfg.EffectiveTokenIssuer(),
			"audience":                   h.cfg.TokenAudience,
			"oauthToken":                 h.cfg.OAuthTokenURL(),
			"managedIdentity":            h.cfg.ManagedIdentityURL(),
			"activeDirectoryEndpointUrl": h.cfg.OAuthTokenURL(),
			"activeDirectoryResourceId":  h.cfg.TokenAudience,
		},
		"management": map[string]string{
			"arm":           managementURL,
			"armHttps":      h.cfg.ManagementTLSURL(),
			"metadata":      managementURL + "/metadata/endpoints",
			"identity":      managementURL + "/metadata/identity",
			"oauth":         h.cfg.OAuthTokenURL(),
			"tenants":       managementURL + "/tenants",
			"providers":     managementURL + "/providers",
			"subscriptions": managementURL + "/subscriptions",
		},
		"resourceManager": map[string]any{
			"endpoint":                   managementURL,
			"resourceManagerEndpointUrl": managementURL,
			"activeDirectoryEndpointUrl": h.cfg.OAuthTokenURL(),
			"activeDirectoryResourceId":  h.cfg.TokenAudience,
			"apiVersions":                []string{"2024-01-01", "2018-02-01"},
			"providers":                  []string{"Microsoft.Resources", "Microsoft.Storage", "Microsoft.KeyVault"},
		},
		"suffixes": map[string]string{
			"storage":   h.cfg.AdvertiseHost + ":" + h.cfg.Blob,
			"keyVault":  h.cfg.AdvertiseHost + ":" + h.cfg.KeyVault,
			"appConfig": h.cfg.AdvertiseHost + ":" + h.cfg.AppConfig,
		},
		"services": map[string]string{
			"blob":       h.cfg.BlobURL(),
			"queue":      h.cfg.QueueURL(),
			"table":      h.cfg.TableURL(),
			"keyVault":   h.cfg.KeyVaultURL(),
			"serviceBus": h.cfg.ServiceBusURL(),
			"appConfig":  h.cfg.AppConfigURL(),
		},
	})
}
