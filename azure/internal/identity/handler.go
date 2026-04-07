package identity

import (
	"net/http"
	"strings"
	"time"

	"tinycloud/internal/auth"
	"tinycloud/internal/config"
	"tinycloud/internal/core/apiversion"
	"tinycloud/internal/httpx"
)

type Handler struct {
	cfg     config.Config
	service *auth.Service
}

func NewHandler(cfg config.Config, service *auth.Service) *Handler {
	return &Handler{cfg: cfg, service: service}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /metadata/identity", h.describeIdentity)
	mux.HandleFunc("GET /metadata/identity/oauth2/token", h.issueManagedIdentityToken)
}

func (h *Handler) describeIdentity(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"name":                 "TinyCloud Managed Identity",
		"tenantId":             h.cfg.TenantID,
		"subscriptionId":       h.cfg.SubscriptionID,
		"issuer":               h.cfg.EffectiveTokenIssuer(),
		"tokenEndpoint":        h.cfg.OAuthTokenURL(),
		"managedIdentityURL":   h.cfg.ManagedIdentityURL(),
		"availableApiVersions": []string{"2018-02-01"},
		"supportedTokenAudiences": []string{
			h.cfg.TokenAudience,
		},
	})
}

func (h *Handler) issueManagedIdentityToken(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Metadata"), "true") {
		httpx.WriteCloudError(w, http.StatusBadRequest, "MissingMetadataHeader", "the Metadata header must be set to true")
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

	audience := strings.TrimSpace(r.URL.Query().Get("resource"))
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	if audience == "" && scope != "" {
		audience = strings.TrimSuffix(strings.TrimSuffix(scope, "/.default"), "/")
	}

	token, err := h.service.IssueToken(audience, "", time.Hour)
	if err != nil {
		httpx.WriteCloudError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
		return
	}
	if audience != "" {
		token.Resource = audience
	}
	if scope != "" {
		token.Scope = scope
	}

	httpx.WriteJSON(w, http.StatusOK, token)
}
