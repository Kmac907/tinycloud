package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"tinycloud/internal/config"
)

func TestIssueTokenReturnsSignedJWT(t *testing.T) {
	t.Parallel()

	cfg := config.FromEnv()
	cfg.TokenKey = "test-signing-key"

	mux := http.NewServeMux()
	NewHandler(NewService(cfg)).Register(mux)

	form := url.Values{
		"resource": []string{"https://storage.azure.com/"},
		"subject":  []string{"alice"},
	}
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body Token
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.AccessToken == "" {
		t.Fatal("access_token is empty")
	}
	if body.Resource != "https://storage.azure.com/" {
		t.Fatalf("resource = %q, want %q", body.Resource, "https://storage.azure.com/")
	}

	claims, err := decodeClaims(body.AccessToken)
	if err != nil {
		t.Fatalf("decodeClaims() error = %v", err)
	}
	if claims["aud"] != "https://storage.azure.com/" {
		t.Fatalf("aud = %v, want %q", claims["aud"], "https://storage.azure.com/")
	}
	if claims["sub"] != "alice" {
		t.Fatalf("sub = %v, want %q", claims["sub"], "alice")
	}
}

func decodeClaims(token string) (map[string]any, error) {
	return DecodeTokenClaims(token)
}
