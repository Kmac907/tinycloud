package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"tinycloud/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /oauth/token", h.issueToken)
}

func (h *Handler) issueToken(w http.ResponseWriter, r *http.Request) {
	audience := strings.TrimSpace(r.FormValue("resource"))
	scope := strings.TrimSpace(r.FormValue("scope"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	if audience == "" && scope == "" {
		audience = h.service.cfg.TokenAudience
	} else if audience == "" {
		audience = normalizeScope(scope)
	}

	token, err := h.service.IssueToken(audience, subject, time.Hour)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}
	if strings.TrimSpace(r.FormValue("resource")) != "" {
		token.Resource = strings.TrimSpace(r.FormValue("resource"))
	}
	if scope != "" {
		token.Scope = scope
	}

	httpx.WriteJSON(w, http.StatusOK, token)
}

func normalizeScope(scope string) string {
	scope = strings.TrimSpace(scope)
	scope = strings.TrimSuffix(scope, "/.default")
	return strings.TrimSuffix(scope, "/")
}

func DecodeTokenClaims(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("token must contain three parts")
	}
	body, err := httpx.DecodeBase64URL(parts[1])
	if err != nil {
		return nil, err
	}

	var claims map[string]any
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}
