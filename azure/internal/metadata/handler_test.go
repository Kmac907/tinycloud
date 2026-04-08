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
		Authentication map[string]string `json:"authentication"`
		Management     map[string]string `json:"management"`
		ResourceMgr    map[string]any    `json:"resourceManager"`
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
	providers, _ := body.ResourceMgr["providers"].([]any)
	if len(providers) != 3 {
		t.Fatalf("len(resourceManager.providers) = %d, want %d", len(providers), 3)
	}
	if body.ResourceMgr["resourceManagerEndpointUrl"] != cfg.ManagementHTTPURL() {
		t.Fatalf("resourceManager.resourceManagerEndpointUrl = %v, want %q", body.ResourceMgr["resourceManagerEndpointUrl"], cfg.ManagementHTTPURL())
	}
}
