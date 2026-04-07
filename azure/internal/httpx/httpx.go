package httpx

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type contextKey string

const requestIDKey contextKey = "request-id"

type CloudErrorResponse struct {
	Error CloudError `json:"error"`
}

type CloudError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteCloudError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, CloudErrorResponse{
		Error: CloudError{
			Code:    code,
			Message: message,
		},
	})
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func NewRequestID(seed []byte) string {
	return hex.EncodeToString(seed)
}

func DecodeBase64URL(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}
