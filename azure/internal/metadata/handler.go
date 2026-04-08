package metadata

import (
	"net/http"
	"strings"

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
	managementURLWithSlash := trailingSlash(managementURL)
	oauthURLWithSlash := trailingSlash(h.cfg.OAuthTokenURL())
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"name":                      "TinyCloud",
		"profile":                   "latest",
		"environment":               "TinyCloudLocal",
		"tenantId":                  h.cfg.TenantID,
		"subscriptionId":            h.cfg.SubscriptionID,
		"activeDirectory":           oauthURLWithSlash,
		"activeDirectoryResourceId": h.cfg.TokenAudience,
		"activeDirectoryGraphResourceId": "https://graph.windows.net/",
		"gallery":                        "https://gallery.azure.com/",
		"management":                     managementURLWithSlash,
		"microsoftGraphResourceId":       "https://graph.microsoft.com/",
		"portal":                         managementURLWithSlash,
		"resourceManager":                managementURLWithSlash,
		"authentication": map[string]any{
			"issuer":                     h.cfg.EffectiveTokenIssuer(),
			"audience":                   h.cfg.TokenAudience,
			"oauthToken":                 h.cfg.OAuthTokenURL(),
			"managedIdentity":            h.cfg.ManagedIdentityURL(),
			"activeDirectoryEndpointUrl": h.cfg.OAuthTokenURL(),
			"activeDirectoryResourceId":  h.cfg.TokenAudience,
		},
		"managementInfo": map[string]string{
			"arm":           managementURL,
			"armHttps":      h.cfg.ManagementTLSURL(),
			"metadata":      managementURL + "/metadata/endpoints",
			"identity":      managementURL + "/metadata/identity",
			"oauth":         h.cfg.OAuthTokenURL(),
			"tenants":       managementURL + "/tenants",
			"providers":     managementURL + "/providers",
			"subscriptions": managementURL + "/subscriptions",
		},
		"resourceManagerInfo": map[string]any{
			"endpoint":                   managementURL,
			"resourceManagerEndpointUrl": managementURL,
			"activeDirectoryEndpointUrl": h.cfg.OAuthTokenURL(),
			"activeDirectoryResourceId":  h.cfg.TokenAudience,
			"apiVersions":                []string{"2024-01-01", "2018-02-01"},
			"providers":                  []string{"Microsoft.Resources", "Microsoft.Storage", "Microsoft.KeyVault", "Microsoft.Network"},
		},
		"endpoints": map[string]any{
			"activeDirectory":                   oauthURLWithSlash,
			"activeDirectoryGraphResourceId":    "https://graph.windows.net/",
			"activeDirectoryResourceId":         h.cfg.TokenAudience,
			"gallery":                           "https://gallery.azure.com/",
			"management":                        managementURLWithSlash,
			"microsoftGraphResourceId":          "https://graph.microsoft.com/",
			"portal":                            managementURLWithSlash,
			"resourceManager":                   managementURLWithSlash,
		},
		"suffixes": map[string]string{
			"storage":   h.cfg.AdvertiseHost + ":" + h.cfg.Blob,
			"keyVault":  h.cfg.AdvertiseHost + ":" + h.cfg.KeyVault,
			"appConfig": h.cfg.AdvertiseHost + ":" + h.cfg.AppConfig,
			"cosmos":    h.cfg.AdvertiseHost + ":" + h.cfg.Cosmos,
			"dns":       h.cfg.DNSAddress(),
			"eventHubs": h.cfg.AdvertiseHost + ":" + h.cfg.EventHubs,
			"keyvaultDns":     h.cfg.AdvertiseHost + ":" + h.cfg.KeyVault,
			"storageEndpoint": h.cfg.AdvertiseHost + ":" + h.cfg.Blob,
		},
		"services": map[string]string{
			"blob":       h.cfg.BlobURL(),
			"queue":      h.cfg.QueueURL(),
			"table":      h.cfg.TableURL(),
			"keyVault":   h.cfg.KeyVaultURL(),
			"serviceBus": h.cfg.ServiceBusURL(),
			"appConfig":  h.cfg.AppConfigURL(),
			"cosmos":     h.cfg.CosmosURL(),
			"dns":        h.cfg.DNSURL(),
			"eventHubs":  h.cfg.EventHubsURL(),
		},
	})
}

func trailingSlash(value string) string {
	return strings.TrimRight(value, "/") + "/"
}
