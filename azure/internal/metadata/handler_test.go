package metadata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tinycloud/internal/config"
)

func TestEndpointsReturnsManagementAndServiceURLs(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()

	mux := http.NewServeMux()
	NewHandler(cfg).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/metadata/endpoints", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		TenantID       string            `json:"tenantId"`
		SubscriptionID string            `json:"subscriptionId"`
		Environment    string            `json:"environment"`
		ResourceManager string           `json:"resourceManager"`
		ActiveDirectory string           `json:"activeDirectory"`
		Authentication map[string]string `json:"authentication"`
		Management     map[string]string `json:"managementInfo"`
		ResourceMgr    map[string]any    `json:"resourceManagerInfo"`
		Endpoints      map[string]string `json:"endpoints"`
		Suffixes       map[string]string `json:"suffixes"`
		Services       map[string]string `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Environment != "TinyCloudLocal" {
		t.Fatalf("environment = %q, want %q", body.Environment, "TinyCloudLocal")
	}
	if body.TenantID != cfg.TenantID {
		t.Fatalf("tenantId = %q, want %q", body.TenantID, cfg.TenantID)
	}
	if body.ResourceManager != cfg.ManagementHTTPURL()+"/" {
		t.Fatalf("resourceManager = %q, want %q", body.ResourceManager, cfg.ManagementHTTPURL()+"/")
	}
	if body.ActiveDirectory != cfg.OAuthTokenURL()+"/" {
		t.Fatalf("activeDirectory = %q, want %q", body.ActiveDirectory, cfg.OAuthTokenURL()+"/")
	}
	if body.Authentication["oauthToken"] != cfg.OAuthTokenURL() {
		t.Fatalf("authentication.oauthToken = %q, want %q", body.Authentication["oauthToken"], cfg.OAuthTokenURL())
	}
	if body.Management["oauth"] != cfg.OAuthTokenURL() {
		t.Fatalf("management.oauth = %q, want %q", body.Management["oauth"], cfg.OAuthTokenURL())
	}
	if body.Management["tenants"] != cfg.ManagementHTTPURL()+"/tenants" {
		t.Fatalf("management.tenants = %q, want %q", body.Management["tenants"], cfg.ManagementHTTPURL()+"/tenants")
	}
	if body.Services["blob"] != cfg.BlobURL() {
		t.Fatalf("services.blob = %q, want %q", body.Services["blob"], cfg.BlobURL())
	}
	if body.Services["appConfig"] != cfg.AppConfigURL() {
		t.Fatalf("services.appConfig = %q, want %q", body.Services["appConfig"], cfg.AppConfigURL())
	}
	if body.Services["cosmos"] != cfg.CosmosURL() {
		t.Fatalf("services.cosmos = %q, want %q", body.Services["cosmos"], cfg.CosmosURL())
	}
	if body.Services["dns"] != cfg.DNSURL() {
		t.Fatalf("services.dns = %q, want %q", body.Services["dns"], cfg.DNSURL())
	}
	if body.Services["eventHubs"] != cfg.EventHubsURL() {
		t.Fatalf("services.eventHubs = %q, want %q", body.Services["eventHubs"], cfg.EventHubsURL())
	}
	if body.Endpoints["resourceManager"] != cfg.ManagementHTTPURL()+"/" {
		t.Fatalf("endpoints.resourceManager = %q, want %q", body.Endpoints["resourceManager"], cfg.ManagementHTTPURL()+"/")
	}
	if body.Endpoints["activeDirectory"] != cfg.OAuthTokenURL()+"/" {
		t.Fatalf("endpoints.activeDirectory = %q, want %q", body.Endpoints["activeDirectory"], cfg.OAuthTokenURL()+"/")
	}
	if body.Authentication["activeDirectoryResourceId"] != cfg.TokenAudience {
		t.Fatalf("authentication.activeDirectoryResourceId = %q, want %q", body.Authentication["activeDirectoryResourceId"], cfg.TokenAudience)
	}
	if body.Suffixes["storage"] != cfg.AdvertiseHost+":"+cfg.Blob {
		t.Fatalf("suffixes.storage = %q, want %q", body.Suffixes["storage"], cfg.AdvertiseHost+":"+cfg.Blob)
	}
	if body.Suffixes["appConfig"] != cfg.AdvertiseHost+":"+cfg.AppConfig {
		t.Fatalf("suffixes.appConfig = %q, want %q", body.Suffixes["appConfig"], cfg.AdvertiseHost+":"+cfg.AppConfig)
	}
	if body.Suffixes["cosmos"] != cfg.AdvertiseHost+":"+cfg.Cosmos {
		t.Fatalf("suffixes.cosmos = %q, want %q", body.Suffixes["cosmos"], cfg.AdvertiseHost+":"+cfg.Cosmos)
	}
	if body.Suffixes["dns"] != cfg.DNSAddress() {
		t.Fatalf("suffixes.dns = %q, want %q", body.Suffixes["dns"], cfg.DNSAddress())
	}
	if body.Suffixes["eventHubs"] != cfg.AdvertiseHost+":"+cfg.EventHubs {
		t.Fatalf("suffixes.eventHubs = %q, want %q", body.Suffixes["eventHubs"], cfg.AdvertiseHost+":"+cfg.EventHubs)
	}
	providers, _ := body.ResourceMgr["providers"].([]any)
	if len(providers) != 4 {
		t.Fatalf("len(resourceManager.providers) = %d, want %d", len(providers), 4)
	}
	if body.ResourceMgr["resourceManagerEndpointUrl"] != cfg.ManagementHTTPURL() {
		t.Fatalf("resourceManager.resourceManagerEndpointUrl = %v, want %q", body.ResourceMgr["resourceManagerEndpointUrl"], cfg.ManagementHTTPURL())
	}
}
