package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tinycloud/internal/httpx"
	"tinycloud/internal/telemetry"
)

func TestRequestIDMiddlewareSetsHeaders(t *testing.T) {
	t.Parallel()

	handler := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"requestID": httpx.RequestID(r.Context())})
	}), withRequestID)

	req := httptest.NewRequest(http.MethodGet, "/_admin/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("x-ms-request-id") == "" {
		t.Fatal("x-ms-request-id header is empty")
	}
	if rec.Header().Get("x-ms-correlation-request-id") == "" {
		t.Fatal("x-ms-correlation-request-id header is empty")
	}
}

func TestRecoveryMiddlewareReturnsCloudError(t *testing.T) {
	t.Parallel()

	logger := telemetry.NewJSONLogger(testWriter{t: t})
	handler := chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), withRequestID, withRecovery(logger))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var body httpx.CloudErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Error.Code != "InternalServerError" {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, "InternalServerError")
	}
}

func TestAPIVersionMiddlewareRejectsARMRequestsWithoutVersion(t *testing.T) {
	t.Parallel()

	handler := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}), withAPIVersion)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/test/resourcegroups", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body httpx.CloudErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Error.Code != "MissingApiVersionParameter" {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, "MissingApiVersionParameter")
	}
}

func TestAPIVersionMiddlewareRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	handler := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}), withAPIVersion)

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/test/resourcegroups?api-version=v1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var body httpx.CloudErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Error.Code != "InvalidApiVersionParameter" {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, "InvalidApiVersionParameter")
	}
}

func TestAPIVersionMiddlewareSkipsAdminRoutes(t *testing.T) {
	t.Parallel()

	handler := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}), withAPIVersion)

	req := httptest.NewRequest(http.MethodGet, "/_admin/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Helper()
	return len(p), nil
}
