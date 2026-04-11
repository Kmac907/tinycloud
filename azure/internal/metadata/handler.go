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
	services := map[string]string{}
	suffixes := map[string]string{}
	if h.cfg.ServiceEnabled(config.ServiceBlob) {
		services["blob"] = h.cfg.BlobURL()
		suffixes["storage"] = h.cfg.AdvertiseHost + ":" + h.cfg.Blob
		suffixes["storageEndpoint"] = h.cfg.AdvertiseHost + ":" + h.cfg.Blob
	}
	if h.cfg.ServiceEnabled(config.ServiceQueue) {
		services["queue"] = h.cfg.QueueURL()
	}
	if h.cfg.ServiceEnabled(config.ServiceTable) {
		services["table"] = h.cfg.TableURL()
	}
	if h.cfg.ServiceEnabled(config.ServiceKeyVault) {
		services["keyVault"] = h.cfg.KeyVaultURL()
		suffixes["keyVault"] = h.cfg.AdvertiseHost + ":" + h.cfg.KeyVault
		suffixes["keyvaultDns"] = h.cfg.AdvertiseHost + ":" + h.cfg.KeyVault
	}
	if h.cfg.ServiceEnabled(config.ServiceServiceBus) {
		services["serviceBus"] = h.cfg.ServiceBusURL()
	}
	if h.cfg.ServiceEnabled(config.ServiceAppConfig) {
		services["appConfig"] = h.cfg.AppConfigURL()
		suffixes["appConfig"] = h.cfg.AdvertiseHost + ":" + h.cfg.AppConfig
	}
	if h.cfg.ServiceEnabled(config.ServiceCosmos) {
		services["cosmos"] = h.cfg.CosmosURL()
		suffixes["cosmos"] = h.cfg.AdvertiseHost + ":" + h.cfg.Cosmos
	}
	if h.cfg.ServiceEnabled(config.ServiceDNS) {
		services["dns"] = h.cfg.DNSURL()
		suffixes["dns"] = h.cfg.DNSAddress()
	}
	if h.cfg.ServiceEnabled(config.ServiceEventHubs) {
		services["eventHubs"] = h.cfg.EventHubsURL()
		suffixes["eventHubs"] = h.cfg.AdvertiseHost + ":" + h.cfg.EventHubs
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"name":                           "TinyCloud",
		"profile":                        "latest",
		"environment":                    "TinyCloudLocal",
		"tenantId":                       h.cfg.TenantID,
		"subscriptionId":                 h.cfg.SubscriptionID,
		"activeDirectory":                oauthURLWithSlash,
		"activeDirectoryResourceId":      h.cfg.TokenAudience,
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
			"activeDirectory":                oauthURLWithSlash,
			"activeDirectoryGraphResourceId": "https://graph.windows.net/",
			"activeDirectoryResourceId":      h.cfg.TokenAudience,
			"gallery":                        "https://gallery.azure.com/",
			"management":                     managementURLWithSlash,
			"microsoftGraphResourceId":       "https://graph.microsoft.com/",
			"portal":                         managementURLWithSlash,
			"resourceManager":                managementURLWithSlash,
		},
		"suffixes": suffixes,
		"services": services,
	})
}

func trailingSlash(value string) string {
	return strings.TrimRight(value, "/") + "/"
}
