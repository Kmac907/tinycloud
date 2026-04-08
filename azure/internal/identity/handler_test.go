package identity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tinycloud/internal/auth"
	"tinycloud/internal/config"
	"tinycloud/internal/httpx"
)

func TestDescribeIdentityReturnsEndpoints(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()

	mux := http.NewServeMux()
	NewHandler(cfg, auth.NewService(cfg)).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/metadata/identity", nil)
	req.Header.Set("Metadata", "true")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["tokenEndpoint"] != cfg.OAuthTokenURL() {
		t.Fatalf("tokenEndpoint = %v, want %q", body["tokenEndpoint"], cfg.OAuthTokenURL())
	}
}

func TestDescribeIdentityRequiresMetadataHeader(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()

	mux := http.NewServeMux()
	NewHandler(cfg, auth.NewService(cfg)).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/metadata/identity", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body httpx.CloudErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Error.Code != "MissingMetadataHeader" {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, "MissingMetadataHeader")
	}
}

func TestManagedIdentityRequiresMetadataHeader(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()

	mux := http.NewServeMux()
	NewHandler(cfg, auth.NewService(cfg)).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body httpx.CloudErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Error.Code != "MissingMetadataHeader" {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, "MissingMetadataHeader")
	}
}

func TestManagedIdentityIssuesToken(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()

	mux := http.NewServeMux()
	NewHandler(cfg, auth.NewService(cfg)).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/", nil)
	req.Header.Set("Metadata", "true")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body auth.Token
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.AccessToken == "" {
		t.Fatal("access_token is empty")
	}
	if body.Resource != "https://management.azure.com/" {
		t.Fatalf("resource = %q, want %q", body.Resource, "https://management.azure.com/")
	}
}
