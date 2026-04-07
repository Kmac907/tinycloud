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
		Management     map[string]string `json:"management"`
		Services       map[string]string `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.TenantID != cfg.TenantID {
		t.Fatalf("tenantId = %q, want %q", body.TenantID, cfg.TenantID)
	}
	if body.Management["oauth"] != cfg.OAuthTokenURL() {
		t.Fatalf("management.oauth = %q, want %q", body.Management["oauth"], cfg.OAuthTokenURL())
	}
	if body.Services["blob"] != cfg.BlobURL() {
		t.Fatalf("services.blob = %q, want %q", body.Services["blob"], cfg.BlobURL())
	}
}
