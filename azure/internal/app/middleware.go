package app

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tinycloud/internal/core/apiversion"
	"tinycloud/internal/httpx"
	"tinycloud/internal/telemetry"
)

func chain(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("x-ms-request-id")
		if requestID == "" {
			requestID = newRequestID()
		}

		ctx := httpx.WithRequestID(r.Context(), requestID)
		w.Header().Set("x-ms-request-id", requestID)
		w.Header().Set("x-ms-correlation-request-id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withLogging(logger *telemetry.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			logger.Info("http request", map[string]any{
				"method":    r.Method,
				"path":      r.URL.Path,
				"remoteIP":  r.RemoteAddr,
				"status":    recorder.status,
				"duration":  time.Since(start).String(),
				"requestID": httpx.RequestID(r.Context()),
			})
		})
	}
}

func withRecovery(logger *telemetry.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", map[string]any{
						"path":      r.URL.Path,
						"requestID": httpx.RequestID(r.Context()),
					})
					httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", "an internal server error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Metadata, x-ms-client-request-id")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withAzureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if clientRequestID := r.Header.Get("x-ms-client-request-id"); clientRequestID != "" {
			w.Header().Set("x-ms-client-request-id", clientRequestID)
		}
		w.Header().Set("x-ms-routing-request-id", "local")
		next.ServeHTTP(w, r)
	})
}

func withAPIVersion(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresAPIVersion(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if _, err := apiversion.Parse(r.URL.Query().Get("api-version")); err != nil {
			code := "InvalidApiVersionParameter"
			message := "the api-version query parameter is invalid"
			if err == apiversion.ErrMissing {
				code = "MissingApiVersionParameter"
				message = "the api-version query parameter is required"
			}
			httpx.WriteCloudError(w, http.StatusBadRequest, code, message)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func requiresAPIVersion(path string) bool {
	for _, prefix := range []string{"/subscriptions", "/providers", "/resourceGroups", "/deployments"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UTC().UnixNano())
	}
	return httpx.NewRequestID(buf)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
